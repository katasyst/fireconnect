# FireConnect (Go)

Single static binary that routes **Claude Code**, **Cursor**, and **Codex/ChatGPT** through the [Fireworks AI](https://fireworks.ai/) inference gateway. Drop-in replacement for the upstream Node.js `fireconnect` CLI — no npm, no native C++ addons, no runtime dependencies.

## Why rewrite?

| Node.js (upstream)                     | Go (this repo)                          |
|----------------------------------------|-----------------------------------------|
| npm + `cross-keychain` (C++ bindings)  | single static binary, zero runtime deps |
| requires Node.js 18+                   | runs anywhere — just copy the binary    |
| `node_modules/` tree                   | `go build` → one file                   |

Functional parity: same CLI flags, same config file formats, same harness on/off/status behavior.

## Supported harnesses

| Harness   | Config target                        | Alias     |
|-----------|--------------------------------------|-----------|
| `claude`  | `~/.claude/settings.json`            |           |
| `cursor`  | `state.vscdb` (SQLite)               |           |
| `codex`   | `~/.codex/config.toml`               | `chatgpt` |

## Quick start

```sh
# sign in
fireconnect login --api-key fw_xxxxx

# enable
fireconnect claude on
fireconnect cursor on
fireconnect codex on        # or: fireconnect chatgpt on

# check status
fireconnect status          # global
fireconnect claude status   # per-harness

# disable (restores backup)
fireconnect claude off
```

## Build

Requires Go 1.22+.

```sh
# current platform
go build -o fireconnect ./cmd/fireconnect

# cross-compile
GOOS=linux   GOARCH=amd64 go build -o fireconnect-linux-amd64   ./cmd/fireconnect
GOOS=darwin  GOARCH=arm64 go build -o fireconnect-darwin-arm64  ./cmd/fireconnect
GOOS=windows GOARCH=amd64 go build -o fireconnect.exe           ./cmd/fireconnect
```

PowerShell (Windows):
```powershell
$env:GOOS="linux";   $env:GOARCH="amd64"; go build -o fireconnect-linux-amd64   ./cmd/fireconnect
$env:GOOS="darwin";  $env:GOARCH="arm64"; go build -o fireconnect-darwin-arm64  ./cmd/fireconnect
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o fireconnect.exe           ./cmd/fireconnect
```

Output is a static binary. Copy it to `PATH` and run.

## Dependencies

Only 3 direct dependencies :

| Package                                                  | Purpose                        | Notes                           |
|----------------------------------------------------------|--------------------------------|---------------------------------|
| [`zalando/go-keyring`](https://github.com/zalando/go-keyring) | OS keychain (Win/Mac/Linux)   | Uses `wincred` / Keychain / `dbus` |
| [`modernc.org/sqlite`](https://modernc.org/sqlite)       | Pure-Go SQLite (Cursor vscdb)  | No CGo, no `libsqlite3`        |
| [`BurntSushi/toml`](https://github.com/BurntSushi/toml)  | TOML parsing (Codex config)    | Maintained by BurntSushi        |

No C compiler needed. No native bindings. Cross-compile from any OS to any OS.

## API key storage

1. OS keychain via `go-keyring` (preferred)
2. Plaintext fallback at `~/.fireconnect/.api-key` (mode `0600`) when keychain unavailable

No keys are hardcoded. No keys are logged or printed.

## License

See upstream [fireworks-ai/fireconnect](https://github.com/fw-ai/fireconnect) for original license.
