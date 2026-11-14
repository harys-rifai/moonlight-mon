# smartmontools

`smartctl.exe` is staged here by `go generate ./agent` (or downloaded by the
release build) and embedded into the Windows agent. This file keeps the
directory present so the agent still builds on machines without the binary;
SMART data is then unavailable while every other metric keeps working.
