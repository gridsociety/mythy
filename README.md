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
| 2    | Modbus TCP/RTU transport + identify/read/set/commands | **complete** |
| 3    | YAML export/import/diff + global --format + parameterized commands | **complete** |
| 4    | G61850 port-504 framing + decode-capture + raw escape hatches polish | not started |

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

## Live operations (Plan 2)

```bash
# Identify a device (default port 502, IANA-registered Modbus TCP)
mythy identify --host 192.0.2.10
# Override port if the device is configured differently
mythy identify --host 192.0.2.10 --port 1502

# Read one or many parameters
mythy read --host 192.0.2.10 MB_address NomeLinea
mythy read --host 192.0.2.10 --scope Read/Measures

# Write parameters (auto-bundled in one edit transaction for *_PARAM,
# direct FC06/FC16 for *_RAM)
mythy set --host 192.0.2.10 MB_address=5 NomeLinea="SAMPLE"

# Invoke any catalog <COMMAND>
mythy command invoke --host 192.0.2.10 MSG_RESET_GUASTI

# Reset shortcuts (faults / counters / measures / defaults)
mythy reset --host 192.0.2.10 faults
mythy reset --host 192.0.2.10 defaults --force   # restore factory

# Reboot (waits for return by default; ~7 s drop, ~2 s outage)
mythy reboot --host 192.0.2.10
mythy reboot --host 192.0.2.10 --no-wait

# Set RTC / network
mythy clock-set --host 192.0.2.10 --at 2026-04-30T12:00:00Z
mythy net-set --host 192.0.2.10 --ip 192.0.2.10 --netmask 255.255.255.0 --gateway 192.0.2.1

# G61850 parser
mythy g61850 list --device PROX-VX0-e        # catalog-only
mythy g61850 invoke --host 192.0.2.10 GetIedName
mythy g61850 invoke --host 192.0.2.10 SetIedName --par1 SAMPLE-IED

# Raw escape hatches (bypass the catalog)
mythy raw read --host 192.0.2.10 --fc 4 --addr 0x143E --qty 5
mythy raw write --host 192.0.2.10 --fc 6 --addr 0x3C2F --value 2

# RTU
mythy identify --serial /dev/ttyUSB0 --baud 19200 --parity N --stopbits 1
```

## Configuration I/O (Plan 3)

```bash
# Snapshot a device's settings to a YAML file
mythy export --host 192.0.2.10 sample-2026-04-30.yaml

# Compare a file to the live device (default human; --format=unified for diff -u style)
mythy diff --host 192.0.2.10 sample-2026-04-30.yaml
mythy diff --host 192.0.2.10 --format=unified sample-2026-04-30.yaml

# Apply changes (one edit transaction wraps every WREG write)
mythy import --host 192.0.2.10 --dry-run sample-2026-04-30.yaml
mythy import --host 192.0.2.10 sample-2026-04-30.yaml

# Cross-device migration: same product → just import. Different product → --force.

# Parameterized device commands (e.g. set the RTC):
mythy command invoke --host 192.0.2.10 SET_RTC \
  --arg RTCDay=30 --arg RTCMonth=4 --arg RTCYear=26 \
  --arg RTCHour=12 --arg RTCMinute=0 --arg RTCSecond=0

# Global output format (also: --json / --yaml aliases, MYTHY_FORMAT env)
mythy identify --host 192.0.2.10 --format=json
mythy read --host 192.0.2.10 --scope Read/Measures --format=yaml
```

## Build

```bash
make build      # → bin/mythy
make test       # run all tests
make lint       # golangci-lint
```
