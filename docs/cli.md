# CLI

Common commands
- `vers status` — list clusters/VMs.
- `vers run` — run a command in a VM.
- `vers execute` — execute a script/command remotely.
- `vers branch [vm-id|alias] --alias <name>` — create a VM branch.
- `vers commit [vm-id|alias] --tags a,b` — commit a VM with tags.
- `vers pause [vm-id|alias]` / `vers resume [vm-id|alias]` — lifecycle.
- `vers kill [vm-id|alias] [-r]` — delete, optionally recursive.
- `vers rename [vm|cluster]` — rename VM or cluster (`-c` for cluster).
- `vers ssh [vm-id|alias]` — connect via SSH.
- `vers ui` — launch interactive TUI (experimental).

Notes
- Commands accept either an ID or alias; if omitted where supported, HEAD is assumed.
- When the SDK expects a `param.Field[T]`, pass with `vers.F(value)`.

