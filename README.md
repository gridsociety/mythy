# mythy

![mythy logo](logo.png)

A Go CLI for Thytronic protection relays (XV10P, NV10P, NA10, …) that speaks
the same wire protocol as the vendor's ThyVisor Windows app: device
identification, catalog browsing, parameter read/write, configuration
export/import, command invocation (including the per-device G61850 parser
functions), over both Modbus TCP and Modbus RTU.

See [the spec](the spec) for the full specification — wire protocol,
catalog layout, CLI surface, architecture and open questions.
