## JSON to TypeScript Interface Generator

A zero-dependency Go CLI that converts any JSON payload into idiomatic TypeScript interfaces, complete with nested types, arrays, unions, and optional fields.

---

### How It Works

- **Recursive walk**: reads a JSON document and walks every nested object and array to produce a full type graph.
- **Optional fields**: any key whose value is `null` is emitted as `field?: any` so the type matches real-world API payloads.
- **Array unions**: heterogeneous arrays produce union types such as `(string | number)[]` automatically.
- **Deterministic output**: fields are sorted alphabetically so re-runs produce diff-friendly output.
- **Root naming**: customize the root interface name with `--name` (default: `Root`).

---

## Setup

### 1. Requirements

- Go 1.21 or higher

### 2. Installation

```bash
git clone https://github.com/fantasywastaken/JSON-to-TypeScript-Interface-Generator.git
cd JSON-to-TypeScript-Interface-Generator
go build -o json2ts
```

---

### Usage

```bash
./json2ts [flags] < input.json
```

From a file with a custom root name:

```bash
$ json2ts --input user.json --name User
export interface Address {
  city: string;
  street: string;
  zip: string;
}

export interface User {
  active: boolean;
  address: Address;
  age: number;
  email: string;
  name: string;
  nickname?: any;
  tags: string[];
}
```

Piping from stdin:

```bash
$ curl -s https://api.example.com/product/1 | json2ts --name Product > product.d.ts
```

Root array becomes a `type` alias:

```bash
$ echo '[{"id":1},{"id":2}]' | json2ts --name Items
export interface Item {
  id: number;
}

export type Items = Item[];
```

---

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | `Root` | Root interface (or type-alias) name |
| `--input` | stdin | Read JSON from this file instead of stdin |
| `--output` | stdout | Write TypeScript to this file instead of stdout |
| `--no-export` | `false` | Omit the `export` keyword (useful for module-internal types) |

---

### Features

- Handles deeply nested objects
- Emits union types for mixed arrays
- Marks nullable fields as optional (`?`)
- Quotes non-identifier keys such as `"user-id"`
- Deterministic, alphabetically sorted fields
- Sensible interface names derived from parent keys (singularized for array items)
- Zero external dependencies, single Go binary
