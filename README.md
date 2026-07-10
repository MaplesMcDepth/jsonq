# jsonq

![CI](https://github.com/MaplesMcDepth/jsonq/actions/workflows/ci.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)


Simple JSON query tool — like jq but you can actually remember the syntax.

## Install

```bash
go install github.com/MaplesMcDepth/jsonq/cmd/jsonq@latest
```

## Commands

### `fmt` — Pretty-print JSON
```bash
cat data.json | jsonq fmt
echo '{"a":1}' | jsonq fmt
```

### `get` — Extract value by path
```bash
jsonq get .users[0].name data.json
jsonq get .count data.json
```

### `keys` — List top-level keys
```bash
jsonq keys data.json
```

### `pluck` — Extract field from array
```bash
jsonq pluck email users.json
```

### `csv` — Convert array of objects to CSV
```bash
jsonq csv products.json > out.csv
```

### `count` — Count array items
```bash
jsonq count items.json
```

### `validate` — Check JSON validity
```bash
echo '{"a":1}' | jsonq validate
```

### `lines` — Output array items one per line
```bash
jsonq lines items.json
```

## Common use cases

### Inspect an API response
```bash
echo '{"users":[{"name":"Ada","active":true}]}' | jsonq get .users[0].name
# Ada
```

### Validate and pretty-print before saving
```bash
cat raw.json | jsonq validate
cat raw.json | jsonq fmt
```

### Pull one field from every record
```bash
echo '[{"email":"ada@example.com"},{"email":"grace@example.com"}]' | jsonq pluck email
# ada@example.com
# grace@example.com
```

### Convert JSON arrays into CSV
```bash
echo '[{"sku":"A1","price":12},{"sku":"B2","price":18}]' | jsonq csv
# price,sku
# 12,A1
# 18,B2
```

## Tips and tricks

- Pipe API output straight into `jsonq` to avoid temporary files.
- Use `jsonq validate` in scripts before handing JSON to another tool.
- `jsonq csv` sorts headers alphabetically, which makes diffs predictable.
- Pair `jsonq lines` with `while read` or `xargs` for quick shell loops.

## Path Syntax

- `.name` — object field
- `.users[0]` — array index
- `.users[0].email` — nested path
