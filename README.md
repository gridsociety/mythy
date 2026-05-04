# mythy

![mythy logo](logo.png)

A command-line client for **Thytronic** protection relays (XV10P, NV10P,
NA10, …). `mythy` speaks the same wire protocol as the vendor's ThyVisor
Windows app — Modbus TCP and Modbus RTU — so you can identify a device,
browse its parameter catalog, read measurements, change settings,
snapshot configurations to YAML, and invoke device commands without
leaving the terminal.

See [the spec](the spec) for the wire protocol, catalog layout, and
architecture in detail.

## What it does

| Capability | Highlights |
|---|---|
| Catalog browsing | Walk the parameter tree, list every leaf, describe one entry; works **without a device** when `--device` is given |
| Live read | Single-name, multi-name, or scoped recursive reads; type-aware decoding for primitives, ENUMs, and 12-field compounds |
| Live write | `name=value` arguments auto-bundle into one Modbus edit transaction; **dotted-path** sub-field syntax for compound types |
| Configuration I/O | Snapshot to YAML, diff against the live device, apply with `--dry-run`/`--force` |
| Commands | Invoke any `<COMMAND>` from the catalog with structured `--arg name=value` parameters |
| G61850 parser | Inspect the device's `Gst61850_Msg` enum, invoke `Get*`/`Set*`/`RestartDevice` over Modbus |
| Convenience wrappers | `reboot`, `reset {faults|counters|measures|defaults}`, `clock-set`, `net-set` |
| Output formats | `human` (default), `json`, `yaml`, `unified` (for diff); `--json`/`--yaml` aliases; `MYTHY_FORMAT` env |
| Transports | Modbus TCP (default port 502) and Modbus RTU on serial / USB-CDC |
| Safety rails | `--force` gate on destructive operations; client-side validation (RANGE bounds, ENUM membership, STRING length) before any write |

## Install

```bash
make build      # → bin/mythy
make test       # run the full test suite
make lint       # golangci-lint
```

`mythy` is a single static binary. No runtime config files.

## Setup

`mythy` reads the parameter catalog from a copy of ThyVisor's
`Templates/` folder. Point at it once per shell session:

```bash
export MYTHY_TEMPLATES="/path/to/ThyVisor/Templates"
```

Or pass `--templates /path/to/Templates` on every command. Locale
defaults to `en` (English DSC strings); override with `--locale it`,
`es`, `ru`, `tr` or set `MYTHY_LOCALE`.

## Usage

### Browse the catalog (no device needed)

```bash
# Print the menu tree for a device
mythy show --device PROX-VX0-e
mythy show --device PROX-VX0-e Set/Base
mythy show --device PROX-VX0-e --include-hidden

# Flat list of every parameter / measurement leaf
mythy list --device PROX-VX0-e
mythy list --device PROX-VX0-e --scope Read/Measures

# Full detail for one entry (TIPO, ENUM values, RANGE bounds, wire info)
mythy describe --device PROX-VX0-e MB_address

# What commands does this product expose?
mythy command list --device PROX-VX0-e
mythy g61850 list --device PROX-VX0-e

# Validate a config YAML against the catalog (no connection)
mythy validate --device PROX-VX0-e my-sample.yaml
```

### Connect and identify

```bash
# Default port 502 (IANA-registered Modbus TCP)
mythy identify --host 192.0.2.10

# Override port if the device is configured differently
mythy identify --host 192.0.2.10 --port 1502

# RTU on serial / USB-CDC
mythy identify --serial /dev/ttyUSB0 --baud 19200 --parity N --stopbits 1
```

`identify` runs the discovery handshake, looks the device up in the
template catalog, and reports the secure-mode state.

### Read

```bash
# One or many named parameters
mythy read --host 192.0.2.10 MB_address NomeLinea

# Recursive scope read (skips hidden / disabled-module DATA)
mythy read --host 192.0.2.10 --scope Read/Measures
mythy read --host 192.0.2.10 --scope Set --include-hidden

# Structured output
mythy read --host 192.0.2.10 --scope Read/Measures --format=json
mythy read --host 192.0.2.10 --scope Read/Measures --yaml
```

### Write

`mythy set name=value …` writes one or more parameters in a single edit
transaction. `*_PARAM` (persistent flash) writes are wrapped in
`START_CHANGE_DB / FC06|16 / END_CHANGE_DB`; `*_RAM` writes go direct.

```bash
# Single scalar
mythy set --host 192.0.2.10 MB_address=5

# Multiple values bundle into one transaction
mythy set --host 192.0.2.10 MB_address=5 NomeLinea="SAMPLE"

# Compound types: dotted-path syntax for sub-fields. mythy reads the
# current compound, mutates the requested sub-fields, and writes the
# whole block back atomically. Multiple sub-fields of the same compound
# coalesce into one read-modify-write.
mythy set --host 192.0.2.10 RELE_K1.Logica=De-energized
mythy set --host 192.0.2.10 RELE_K1.Logica=Energized RELE_K1.Modo=NormalOpen
mythy set --host 192.0.2.10 EnF81_TSc.Valore=2000
```

Values are validated client-side against the catalog (TIPO width,
`<RANGE>` bounds, ENUM label membership, STRING length) before any
Modbus write is issued.

### Invoke device commands

```bash
# Trigger any catalog <COMMAND> by name
mythy command invoke --host 192.0.2.10 MSG_RESET_GUASTI

# Parameterized commands (the WREG-with-parts variety) take --arg
mythy command invoke --host 192.0.2.10 SET_RTC \
  --arg RTCDay=30 --arg RTCMonth=4 --arg RTCYear=26 \
  --arg RTCHour=12 --arg RTCMinute=0 --arg RTCSecond=0

# Common command shortcuts
mythy reboot --host 192.0.2.10                      # waits for the device to return
mythy reboot --host 192.0.2.10 --no-wait
mythy reset --host 192.0.2.10 faults
mythy reset --host 192.0.2.10 counters
mythy reset --host 192.0.2.10 measures
mythy reset --host 192.0.2.10 defaults --force      # restore factory settings
mythy clock-set --host 192.0.2.10 --at 2026-04-30T12:00:00Z
mythy net-set --host 192.0.2.10 --ip 192.0.2.10 --netmask 255.255.255.0 --gateway 192.0.2.1
```

### G61850 parser

The device exposes a small RPC parser for actions like renaming the IED,
restarting the device, or reading IEC 61850 metadata. `mythy g61850 list`
shows what's supported on a given product (per the template's
`Gst61850_Msg` enum); `mythy g61850 invoke` calls it.

```bash
mythy g61850 list --device PROX-VX0-e        # no connection needed
mythy g61850 invoke --host 192.0.2.10 GetIedName
mythy g61850 invoke --host 192.0.2.10 SetIedName --par1 SAMPLE-IED

# Destructive functions (WriteCid, ResetCid, ResetAll) require --force.
# Use with care.
mythy g61850 invoke --host 192.0.2.10 ResetAll --force
```

### Configuration I/O

```bash
# Snapshot a device's settings to YAML
mythy export --host 192.0.2.10 sample.yaml

# Compare a file to the live device
mythy diff --host 192.0.2.10 sample.yaml                      # human table
mythy diff --host 192.0.2.10 --format=unified sample.yaml     # diff -u style
mythy diff --host 192.0.2.10 --json sample.yaml               # structured

# Preview, then apply (one edit transaction wraps every change)
mythy import --host 192.0.2.10 --dry-run sample.yaml
mythy import --host 192.0.2.10 sample.yaml

# Cross-device migration: same product → just import.
# Different product → pass --force to skip the product-mismatch check.
```

The YAML schema is documented in [the spec § 4](the spec). Read-only,
hidden, and module-disabled DATA are excluded by default; `--include-readonly`
and `--include-hidden` widen the export.

### Raw escape hatches

When you need to bypass the catalog (debugging, reverse engineering,
hitting a register the catalog doesn't surface):

```bash
mythy raw read --host 192.0.2.10 --fc 4 --addr 0x143E --qty 5
mythy raw write --host 192.0.2.10 --fc 6 --addr 0x3C2F --value 2
```

## Output formats

Every command that produces structured output supports `--format`:

```bash
mythy identify --host 192.0.2.10 --format=json
mythy read --host 192.0.2.10 --scope Read/Measures --format=yaml
mythy diff --host 192.0.2.10 file.yaml --format=unified
```

Aliases: `--json` ≡ `--format=json`, `--yaml` ≡ `--format=yaml`. Falls
back to the `MYTHY_FORMAT` environment variable, or `human` if neither
is set.

## Connection flags (apply to every live command)

| Flag | Default | Notes |
|---|---|---|
| `--host` | – | TCP host (e.g. `192.0.2.10`) |
| `--port` | `502` | TCP port; IANA-registered Modbus TCP |
| `--serial` | – | RTU device path (e.g. `/dev/ttyUSB0`); auto-selects RTU |
| `--baud` | `19200` | RTU baud rate |
| `--parity` | `N` | RTU parity: `N`, `E`, or `O` |
| `--stopbits` | `1` | RTU stop bits |
| `--unit-id` | `1` | Modbus unit ID |
| `--request-timeout` | `2s` | per-request timeout |
| `--connect-timeout` | `5s` | TCP connect timeout |
| `--retries` | `2` | transient-error retries on reads (writes never retry) |

## Help

Every subcommand has built-in `--help`:

```bash
mythy --help
mythy set --help
mythy g61850 invoke --help
```

## Project layout

```
cmd/mythy/      CLI (cobra)
pkg/catalog/    Vendor template parser (Codifica.xml + per-device XML)
pkg/codec/      TIPO ↔ register words (primitives, enums, compounds)
pkg/transport/  Modbus TCP / RTU clients
pkg/session/    High-level API: Connect, Identify, Read, Set, Command, …
pkg/configio/   YAML export / import / diff / apply
testdata/       Synthetic catalog fixture for tests
```
