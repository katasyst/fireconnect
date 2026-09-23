# FireConnect (Go)

Single static binary that routes **Claude Code**, **Cursor**, and **Codex/ChatGPT** through [Fireworks AI](https://fireworks.ai/) or **Azure OpenAI**. Drop-in replacement for the upstream Node.js CLI — no npm, no native addons, no runtime dependencies.

Fireworks AI is **ZDR by default** — no prompts, completions, or request logs are stored after the response is returned.

## Install

```sh
git clone https://github.com/katasyst/fireconnect.git
cd fireconnect
go install ./cmd/fireconnect
```

Binary lands in `$GOPATH/bin/fireconnect` (should already be in PATH).

## Sign in (Fireworks)

```sh
fireconnect login --api-key fw_YOUR_KEY_HERE
```

Key stored at `~/.fireconnect/.api-key` (mode `0600`). No OS keychain, no env vars.

| OS | Path |
|---|---|
| Linux / macOS | `~/.fireconnect/.api-key` |
| Windows | `%USERPROFILE%\.fireconnect\.api-key` |

## Sign in (Azure OpenAI)

```sh
fireconnect azure-login --api-key YOUR_AZURE_KEY --base-url https://xxx.services.ai.azure.com/openai/v1
```

Credentials stored in `~/.fireconnect/config.json` (mode `0600`). Only needs to be done once.

## Browse models (Fireworks)

```sh
fireconnect model list                  # all models & routers
fireconnect model list --search kimi    # filter
fireconnect model list --refresh        # force refresh (cache: 1 hour)
```

## Enable a harness

**Claude Code** (Fireworks only):
```sh
fireconnect claude on
fireconnect claude on --opus glm-5p3-flash --sonnet glm-5p3-flash --haiku glm-5p3-flash
```

**Cursor** (Fireworks only, quit Cursor first):
```sh
fireconnect cursor on
```

**Codex / ChatGPT** (Fireworks):
```sh
fireconnect codex on
```

**Codex / ChatGPT** (Azure OpenAI — direct, no Fireworks involved):
```sh
fireconnect codex on --azure --model gpt-6-luna
```

Check status:
```sh
fireconnect status          # global (shows both Fireworks + Azure)
fireconnect codex status    # per-harness (shows provider + model + endpoint)
```

Disable (restores previous settings):
```sh
fireconnect codex off
```

## Provider routing

| Harness | Fireworks | Azure OpenAI | Why |
|---|---|---|---|
| `claude` | **Yes** | No | Claude Code speaks Anthropic API format. Fireworks translates server-side. Azure OpenAI doesn't. |
| `cursor` | **Yes** | Not yet | |
| `codex` | **Yes** | **Yes** | Codex speaks OpenAI format natively. Azure OpenAI is OpenAI-compatible. Direct connection. |

## Claude Code slot reference

| Flag | Env var | Default |
|---|---|---|
| `--model` | Top-level `model` key | `kimi-fast-latest` |
| `--opus` | `ANTHROPIC_DEFAULT_OPUS_MODEL` | `kimi-fast-latest` |
| `--sonnet` | `ANTHROPIC_DEFAULT_SONNET_MODEL` | `glm-fast-latest` |
| `--haiku` | `ANTHROPIC_DEFAULT_HAIKU_MODEL` | `deepseek-flash-latest` |
| `--fable` | `ANTHROPIC_DEFAULT_FABLE_MODEL` | `deepseek-pro-latest` |
| `--subagent` | `CLAUDE_CODE_SUBAGENT_MODEL` | (unset unless specified) |

All slots get `[1m]` appended — tells Claude Code to use the 1M context window. The Fireworks gateway strips it.

## Cross-platform build

Requires Go 1.22+. Output is a single static binary.

```sh
GOOS=linux   GOARCH=amd64 go build -o fireconnect-linux-amd64   ./cmd/fireconnect
GOOS=darwin  GOARCH=arm64 go build -o fireconnect-darwin-arm64  ./cmd/fireconnect
GOOS=windows GOARCH=amd64 go build -o fireconnect.exe           ./cmd/fireconnect
```

## Update

```sh
cd fireconnect
git pull
go install ./cmd/fireconnect
```

## Dependencies

2 direct dependencies:

| Package | Purpose |
|---|---|
| [`modernc.org/sqlite`](https://modernc.org/sqlite) | Pure-Go SQLite for Cursor's `state.vscdb` |
| [`BurntSushi/toml`](https://github.com/BurntSushi/toml) | TOML parsing for Codex config |

No C compiler. No native bindings. No OS keychain.

## License

See upstream [fireworks-ai/fireconnect](https://github.com/fw-ai/fireconnect) for original license.
