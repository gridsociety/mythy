# mythy

![mythy logo](logo.png)

A Go CLI for Thytronic protection relays (XV10P, NV10P, NA10, …) that speaks
the same wire protocol as the vendor's ThyVisor Windows app: device
identification, catalog browsing, parameter read/write, configuration
export/import, command invocation (including the per-device G61850 parser
functions), over both Modbus TCP and Modbus RTU.

See [the spec](the spec) for the full specification — wire protocol,
catalog layout, CLI surface, architecture and open questions.

## Status

| Plan | Scope | Status |
|------|-------|--------|
| 1    | Catalog parser + codec primitives + catalog-only CLI | **complete** |
| 2    | Modbus TCP/RTU transport + identify/read/set/commands | not started |
| 3    | YAML export/import/diff | not started |
| 4    | G61850 parser + raw I/O + decode-capture | not started |

## Quickstart (Plan 1: catalog-only)

`mythy` reads the catalog from a copy of ThyVisor's `Templates/` folder.
Point at it via `--templates` or `MYTHY_TEMPLATES`:

```bash
export MYTHY_TEMPLATES="/path/to/ThyVisor/Templates"

# Browse the parameter tree of a specific device
mythy show --device PROX-VX0-e
mythy show --device PROX-VX0-e Set/Base
mythy show --device PROX-VX0-e --include-hidden

# Flat list of every leaf
mythy list --device PROX-VX0-e
mythy list --device PROX-VX0-e --scope Read/Measures

# Full detail for one entry
mythy describe --device PROX-VX0-e MB_address

# What commands does the device expose?
mythy command list --device PROX-VX0-e
mythy g61850 list --device PROX-VX0-e

# Validate a config YAML against the catalog (no connection)
mythy validate --device PROX-VX0-e my-sample.yaml
```

## Build

```bash
make build      # → bin/mythy
make test       # run all tests
make lint       # golangci-lint
```
