package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

type tsField struct {
	Name     string
	Type     string
	Optional bool
}

type tsInterface struct {
	Name   string
	Fields []tsField
}

type generator struct {
	Interfaces []tsInterface
	nameCounts map[string]int
	exportKind string
}

var reIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
var reNonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

func toPascalCase(s string) string {
	parts := reNonAlnum.Split(s, -1)
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	if b.Len() == 0 {
		return "Item"
	}
	return b.String()
}

func toSingular(s string) string {
	if strings.HasSuffix(s, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "sses") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && len(s) > 1 && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

func (g *generator) uniqueName(base string) string {
	base = toPascalCase(base)
	if _, taken := g.nameCounts[base]; !taken {
		g.nameCounts[base] = 1
		return base
	}
	g.nameCounts[base]++
	return fmt.Sprintf("%s%d", base, g.nameCounts[base])
}

func (g *generator) inferType(fieldName string, v interface{}) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return g.inferArray(fieldName, t)
	case map[string]interface{}:
		name := g.uniqueName(fieldName)
		iface := g.buildInterface(name, t)
		g.Interfaces = append(g.Interfaces, iface)
		return name
	default:
		return "any"
	}
}

func (g *generator) inferArray(fieldName string, arr []interface{}) string {
	if len(arr) == 0 {
		return "any[]"
	}
	itemName := toSingular(fieldName)
	if itemName == fieldName {
		itemName = fieldName + "Item"
	}
	seen := map[string]struct{}{}
	var types []string
	for _, item := range arr {
		t := g.inferType(itemName, item)
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		types = append(types, t)
	}
	if len(types) == 1 {
		return types[0] + "[]"
	}
	sort.Strings(types)
	return "(" + strings.Join(types, " | ") + ")[]"
}

func (g *generator) buildInterface(name string, obj map[string]interface{}) tsInterface {
	iface := tsInterface{Name: name}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := obj[k]
		if v == nil {
			iface.Fields = append(iface.Fields, tsField{Name: k, Type: "any", Optional: true})
			continue
		}
		iface.Fields = append(iface.Fields, tsField{Name: k, Type: g.inferType(k, v)})
	}
	return iface
}

func (g *generator) render() string {
	var b strings.Builder
	for i := len(g.Interfaces) - 1; i >= 0; i-- {
		iface := g.Interfaces[i]
		if g.exportKind != "" {
			b.WriteString(g.exportKind)
			b.WriteString(" ")
		}
		b.WriteString("interface ")
		b.WriteString(iface.Name)
		b.WriteString(" {\n")
		for _, f := range iface.Fields {
			name := f.Name
			if !reIdentifier.MatchString(name) {
				name = fmt.Sprintf("%q", name)
			}
			opt := ""
			if f.Optional {
				opt = "?"
			}
			b.WriteString(fmt.Sprintf("  %s%s: %s;\n", name, opt, f.Type))
		}
		b.WriteString("}\n")
		if i > 0 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func main() {
	rootName := flag.String("name", "Root", "root interface name")
	inputPath := flag.String("input", "", "input JSON file (default: stdin)")
	outputPath := flag.String("output", "", "output file (default: stdout)")
	noExport := flag.Bool("no-export", false, "omit the export keyword")
	flag.Parse()

	var data []byte
	var err error
	if *inputPath == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(*inputPath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot read input:", err)
		os.Exit(1)
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		fmt.Fprintln(os.Stderr, "error: invalid JSON:", err)
		os.Exit(1)
	}

	g := &generator{nameCounts: map[string]int{}, exportKind: "export"}
	if *noExport {
		g.exportKind = ""
	}

	var trailing string
	switch v := raw.(type) {
	case map[string]interface{}:
		name := g.uniqueName(*rootName)
		iface := g.buildInterface(name, v)
		g.Interfaces = append(g.Interfaces, iface)
	case []interface{}:
		arrType := g.inferArray(*rootName, v)
		if *noExport {
			trailing = fmt.Sprintf("type %s = %s;\n", *rootName, arrType)
		} else {
			trailing = fmt.Sprintf("export type %s = %s;\n", *rootName, arrType)
		}
	default:
		fmt.Fprintln(os.Stderr, "error: root value must be a JSON object or array")
		os.Exit(1)
	}

	out := g.render()
	if trailing != "" {
		if out != "" {
			out += "\n"
		}
		out += trailing
	}

	if *outputPath == "" {
		fmt.Print(out)
		return
	}
	if err := os.WriteFile(*outputPath, []byte(out), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot write output:", err)
		os.Exit(1)
	}
}
