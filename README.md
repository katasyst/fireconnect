# FireConnect (Go)

Single static binary that routes **Claude Code**, **Cursor**, and **Codex/ChatGPT** through [Fireworks AI](https://fireworks.ai/). Drop-in replacement for the upstream Node.js CLI — no npm, no native addons, no runtime dependencies.

Fireworks AI is **ZDR by default** — no prompts, completions, or request logs are stored after the response is returned.

<img width="1842" height="826" alt="image" src="https://github.com/user-attachments/assets/2143d618-2c0b-42d4-85de-d78398ab407d" />


## Install

```sh
git clone https://github.com/katasyst/fireconnect.git
cd fireconnect
go install ./cmd/fireconnect
```

Binary lands in `$GOPATH/bin/fireconnect` (should already be in PATH).

## Sign in

Get your API key from [fireworks.ai/settings](https://app.fireworks.ai/settings/users/api-keys):

```sh
fireconnect login --api-key fw_YOUR_KEY_HERE
```

This creates `~/.fireconnect/.api-key` (file mode `0600`, owner-only read/write).
That's the only place the key is stored — no OS keychain, no environment variables.

> **Where exactly?**
> | OS | Path |
> |---|---|
> | Linux / macOS | `~/.fireconnect/.api-key` |
> | Windows | `%USERPROFILE%\.fireconnect\.api-key` |
>
> Same security model as `~/.ssh/id_rsa` — just a file with restricted permissions.
> The key is never printed to terminal or logged anywhere.

## Browse models

```sh
fireconnect model list                  # all models & routers
fireconnect model list --search kimi    # filter
fireconnect model list --refresh        # force refresh (cache: 1 hour)
```

## Enable a harness

**Claude Code:**
```sh
fireconnect claude on
fireconnect claude on --opus glm-5p3-flash --sonnet glm-5p3-flash --haiku glm-5p3-flash --fable glm-5p3-flash --subagent glm-5p3-flash
```

**Cursor** (quit Cursor first):
```sh
fireconnect cursor on
```

**Codex / ChatGPT:**
```sh
fireconnect codex on
```

Check status:
```sh
fireconnect status          # global
fireconnect claude status   # per-harness
```

Disable (restores previous settings):
```sh
fireconnect claude off
```

## Claude Code slot reference

| Flag | Env var it sets | Default |
|---|---|---|
| `--model` | Top-level `model` key | `kimi-fast-latest` |
| `--opus` | `ANTHROPIC_DEFAULT_OPUS_MODEL` | `kimi-fast-latest` |
| `--sonnet` | `ANTHROPIC_DEFAULT_SONNET_MODEL` | `glm-fast-latest` |
| `--haiku` | `ANTHROPIC_DEFAULT_HAIKU_MODEL` | `deepseek-flash-latest` |
| `--fable` | `ANTHROPIC_DEFAULT_FABLE_MODEL` | `deepseek-pro-latest` |
| `--subagent` | `CLAUDE_CODE_SUBAGENT_MODEL` | (unset unless specified) |

All slots get `[1m]` appended automatically — tells Claude Code to use the 1M context window. The Fireworks gateway strips it on the wire.

## Supported harnesses

| Harness | Config target | Alias |
|---|---|---|
| `claude` | `~/.claude/settings.json` | |
| `cursor` | `state.vscdb` (SQLite) | |
| `codex` | `~/.codex/config.toml` | `chatgpt` |

## Cross-platform build

Requires Go 1.22+. 

```sh
# current OS
go build -o fireconnect ./cmd/fireconnect

# cross-compile
GOOS=linux   GOARCH=amd64 go build -o fireconnect-linux-amd64   ./cmd/fireconnect
GOOS=darwin  GOARCH=arm64 go build -o fireconnect-darwin-arm64  ./cmd/fireconnect
GOOS=windows GOARCH=amd64 go build -o fireconnect.exe           ./cmd/fireconnect
```

PowerShell:
```powershell
$env:GOOS="linux";   $env:GOARCH="amd64"; go build -o fireconnect-linux-amd64   ./cmd/fireconnect
$env:GOOS="darwin";  $env:GOARCH="arm64"; go build -o fireconnect-darwin-arm64  ./cmd/fireconnect
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o fireconnect.exe           ./cmd/fireconnect
```

## Update

```sh
cd fireconnect
git pull
go install ./cmd/fireconnect
```


## License

See upstream [fireworks-ai/fireconnect](https://github.com/fw-ai/fireconnect) for original license.
