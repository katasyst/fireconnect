# FireConnect (Go)

Single go binary that routes **Claude Code**, **Cursor**, and **Codex/ChatGPT** through the [Fireworks AI](https://fireworks.ai/) inference gateway. Drop-in replacement for the upstream Node.js `fireconnect` CLI — no npm, no native C++ addons, no runtime dependencies.

<img width="1842" height="826" alt="image" src="https://github.com/user-attachments/assets/2143d618-2c0b-42d4-85de-d78398ab407d" />

## Setup guide

### 1. Install

**Option A — from source (recommended):**
```sh
git clone https://github.com/katasyst/fireconnect.git
cd fireconnect
go install ./cmd/fireconnect
```
Binary goes to `$GOPATH/bin/fireconnect` (already in PATH if Go is set up).

**Option B — build manually:**
```sh
go build -o fireconnect ./cmd/fireconnect    # Linux/Mac
go build -o fireconnect.exe ./cmd/fireconnect # Windows
```
Copy the binary somewhere in your PATH.

### 2. Sign in

Get your API key from [Fireworks AI](https://app.fireworks.ai/settings/users/api-keys), then:

```sh
fireconnect login --api-key fw_YOUR_KEY_HERE
```

Key is stored in your OS keychain (Windows Credential Manager / macOS Keychain / Linux Secret Service). Falls back to `~/.fireconnect/.api-key` (mode 0600) if keychain is unavailable.

### 3. Browse available models

```sh
fireconnect model list                  # show all models & routers
fireconnect model list --search kimi    # filter by name
fireconnect model list --search glm     # filter by name
fireconnect model list --refresh        # force refresh (cache TTL: 1 hour)
fireconnect model list --json           # machine-readable output
```

### 4. Enable a harness

**Claude Code** (most common):
```sh
# quick start — uses default models
fireconnect claude on

# pick a specific main model
fireconnect claude on --model kimi-fast-latest

# customize every slot
fireconnect claude on \
  --opus glm-5p3-flash \
  --sonnet glm-5p3-flash \
  --haiku glm-5p3-flash \
  --fable glm-5p3-flash \
  --subagent glm-5p3-flash
```

PowerShell (Windows) — all on one line:
```powershell
fireconnect claude on --opus glm-5p3-flash --sonnet glm-5p3-flash --haiku glm-5p3-flash --fable glm-5p3-flash --subagent glm-5p3-flash
```

**Cursor IDE:**
```sh
# quit Cursor first, then:
fireconnect cursor on
# reopen Cursor
```

**Codex / ChatGPT:**
```sh
fireconnect codex on          # or: fireconnect chatgpt on
```

### 5. Verify

```sh
fireconnect status            # global sign-in state
fireconnect claude status     # Claude slot mapping & provider
fireconnect cursor status     # Cursor connection state
fireconnect codex status      # Codex connection state
```

### 6. Disable (restores your previous settings)

```sh
fireconnect claude off
fireconnect cursor off
fireconnect codex off
```

## Claude Code slot reference

| Flag          | What it controls                        | Default                    |
|---------------|-----------------------------------------|----------------------------|
| `--model`     | Main model (top-level `model` key)      | `kimi-fast-latest`         |
| `--opus`      | `ANTHROPIC_DEFAULT_OPUS_MODEL`          | `kimi-fast-latest`         |
| `--sonnet`    | `ANTHROPIC_DEFAULT_SONNET_MODEL`        | `glm-fast-latest`          |
| `--haiku`     | `ANTHROPIC_DEFAULT_HAIKU_MODEL`         | `deepseek-flash-latest`    |
| `--fable`     | `ANTHROPIC_DEFAULT_FABLE_MODEL`         | `deepseek-pro-latest`      |
| `--subagent`  | `CLAUDE_CODE_SUBAGENT_MODEL`            | (unset unless specified)   |

All slots get `[1m]` appended automatically — Claude Code uses this to size the context window to 1M tokens. The Fireworks gateway strips it on the wire.

Re-running `on` with different flags overwrites only the slots you specify.

## Supported harnesses

| Harness   | Config target                        | Alias     |
|-----------|--------------------------------------|-----------|
| `claude`  | `~/.claude/settings.json`            |           |
| `cursor`  | `state.vscdb` (SQLite)               |           |
| `codex`   | `~/.codex/config.toml`               | `chatgpt` |

## Cross-platform build

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

Output is a static binary. No runtime dependencies.

## Dependencies

Only 3 direct dependencies — all well-audited, widely used:

| Package                                                  | Purpose                        | Notes                           |
|----------------------------------------------------------|--------------------------------|---------------------------------|
| [`zalando/go-keyring`](https://github.com/zalando/go-keyring) | OS keychain (Win/Mac/Linux)   | Uses `wincred` / Keychain / `dbus` |
| [`modernc.org/sqlite`](https://modernc.org/sqlite)       | Pure-Go SQLite (Cursor vscdb)  | No CGo, no `libsqlite3`        |
| [`BurntSushi/toml`](https://github.com/BurntSushi/toml)  | TOML parsing (Codex config)    | Maintained by BurntSushi        |

No C compiler needed. No native bindings. Cross-compile from any OS to any OS.

## API key storage

1. OS keychain via `go-keyring` (preferred — Windows Credential Manager / macOS Keychain / Linux Secret Service)
2. Plaintext fallback at `~/.fireconnect/.api-key` (mode `0600`) when keychain unavailable

No keys are hardcoded. No keys are logged or printed.

## CLI reference

```
fireconnect login --api-key <key>     Sign in
fireconnect logout                    Sign out
fireconnect status                    Global sign-in state

fireconnect model list [--search Q]   Browse model catalog
fireconnect model list --refresh      Force refresh catalog

fireconnect <harness> on [flags]      Enable Fireworks routing
fireconnect <harness> off             Restore previous settings
fireconnect <harness> status          Show connection state

fireconnect --version                 Print version
fireconnect help                      Full help
fireconnect help <harness>            Harness-specific help
```

## License

See upstream [fireworks-ai/fireconnect](https://github.com/fw-ai/fireconnect) for original license.
