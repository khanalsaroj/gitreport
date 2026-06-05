# AI Providers

gitreport selects an AI provider automatically. It detects which providers are
usable on the current machine, uses the most preferred one, and falls back to
the next if a provider is unavailable, rate-limited, or fails.

## Priority order

The default order, best first:

1. **Claude Code** — the local `claude` CLI, if installed
2. **OpenAI** — API key
3. **Google Gemini** — API key
4. **xAI Grok** — API key
5. **OpenRouter** — API key (the universal fallback)

Inspect what is detected on your machine:

```bash
gitreport providers
```

```
PROVIDER     NAME           STATUS               DETAIL
claude-code  Claude Code    available (primary)  ready
openai       OpenAI         unavailable          openai: API key not configured
gemini       Google Gemini  unavailable          gemini: API key not configured
grok         xAI Grok       unavailable          grok: API key not configured
openrouter   OpenRouter     available            ready
```

## How selection works

- **Detection** is cheap and local: a provider is *available* if its credential
  is configured (HTTP providers) or its binary is on `PATH` (Claude Code). No
  network calls or paid requests are made during detection.
- **Fallback** happens at stream time. The chain commits to the first provider
  that produces output. If a provider fails *before* emitting any text
  (unavailable, bad key, rate limit), the chain retries transient errors
  (HTTP 429/5xx and network failures) with backoff, then moves to the next
  provider. Once output has started, errors are surfaced rather than masked by a
  silent switch.
- **Logging** goes to stderr (so stdout stays clean for the report). Use
  `--verbose`, or `GITREPORT_LOG=debug|info|warn|error`, to see selection,
  fallback, and retry decisions.

## Configuration

All configuration lives in `~/.gitreport/setting.json`. Everything is optional —
if Claude Code is installed, gitreport works with no keys at all.

```json
{
  "OPENAI_API_KEY": "sk-or-...",
  "OPENAI_BASE_URL": "https://openrouter.ai/api/v1/chat/completions",
  "OPENAI_MODEL": "nvidia/nemotron-3-super-120b-a12b:free",

  "priority": ["claude-code", "openai", "gemini", "grok", "openrouter"],

  "providers": {
    "claude-code": { "enabled": true, "model": "claude-opus-4-8" },
    "openai":      { "api_key": "sk-...",     "model": "gpt-4o-mini" },
    "gemini":      { "api_key": "...",        "model": "gemini-2.0-flash" },
    "grok":        { "api_key": "...",        "model": "grok-2-latest" },
    "openrouter":  { "api_key": "sk-or-...",  "model": "..." }
  }
}
```

- The flat **`OPENAI_*`** fields are the legacy single-endpoint configuration and
  remain fully supported. They map to whichever provider their base URL targets
  (OpenRouter by default), and serve as the fallback tier.
- **`priority`** sets the preference order. A partial list is completed with the
  remaining providers, so listing a favorite still allows fallback.
- **`providers.<id>`** holds per-provider `api_key`, `base_url`, `model`, and an
  `enabled` toggle.

### Environment variables

Environment variables override `setting.json`:

| Variable | Effect |
|----------|--------|
| `GITREPORT_PROVIDER` | Force a single provider (no fallback) |
| `GITREPORT_PROVIDER_PRIORITY` | Comma-separated priority order |
| `GITREPORT_OPENAI_API_KEY` | OpenAI API key |
| `GEMINI_API_KEY` / `GOOGLE_API_KEY` | Gemini API key |
| `XAI_API_KEY` / `GROK_API_KEY` | Grok API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `OPENAI_API_KEY` / `OPENAI_BASE_URL` / `OPENAI_MODEL` | Legacy single endpoint |
| `GITREPORT_LOG` | Log level: `debug`, `info`, `warn`, `error` |

The CLI flags `--provider <id>` and `--verbose` override per invocation.

## Feasibility analysis and limitations

This was assessed against the actual tools installed on a developer machine.

### Claude Code — supported, with caveats

The `claude` CLI supports non-interactive use (`claude -p --output-format json`),
so gitreport drives it as a subprocess and reuses the CLI's own authentication
(your Claude subscription or API key). No key is configured in gitreport.

Caveats, by design:

- **Requires the `claude` CLI** to be installed and authenticated. Detection is
  by binary presence only; an unauthenticated CLI is discovered lazily at stream
  time and triggers fallback.
- **Not token-streamed.** In JSON output mode the CLI returns the full answer at
  once, so the report appears in one block rather than streaming token by token.
- **Higher latency and cost** than a raw API call, because each invocation starts
  a CLI session. gitreport passes `--system-prompt` (replacing the heavyweight
  default agent prompt) and runs in a neutral directory to minimize this, but it
  remains heavier than a direct HTTP request.

### OpenAI — API key only

A **ChatGPT Plus/Pro subscription cannot be used programmatically.** That is a
web product with no API. Only an OpenAI **API key** (billed separately, via the
platform) works. gitreport therefore detects OpenAI by API key, not by any
"subscription" signal — there is no supported way to detect or use a ChatGPT
subscription.

### Gemini and Grok — API key via OpenAI-compatible endpoints

Both expose OpenAI-compatible chat-completions endpoints, so they reuse the same
HTTP provider with a different base URL and key. A `gemini` CLI also exists, but
the HTTP API is used instead because it is more reliable and supports true token
streaming. A CLI-backed provider could be added later (see below) if desired.

### OpenRouter — the universal fallback

Already supported and unchanged. It aggregates many models (including free
tiers), which makes it a sensible always-available fallback.

## Adding a provider

Most providers are data-driven. To add another OpenAI-compatible service, add one
entry to `specs` in [`internal/ai/specs.go`](../internal/ai/specs.go):

```go
ProviderTogether: {
    id:           ProviderTogether,
    kind:         kindHTTP,
    displayName:  "Together AI",
    envKeys:      []string{"TOGETHER_API_KEY"},
    defaultURL:   "https://api.together.xyz/v1/chat/completions",
    defaultModel: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
},
```

and include its ID in `DefaultPriority`. No other code changes are required.
Providers with a bespoke protocol (like Claude Code) implement the `Provider`
interface directly; see [`claudecode.go`](../internal/ai/claudecode.go).
