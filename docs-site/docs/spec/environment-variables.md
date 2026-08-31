---
title: Environment variables
---

# Environment variables

The environment variables that configure the reviewer. Which one you need depends on the
review mode: `batch` mode talks to Gemini directly, `dependency` mode talks to a LiteLLM
(LLM, large language model) proxy. `LOG_LEVEL` is optional in both.

## Reference

| Variable | Required | Default | Description |
|---|---|---|---|
| `GEMINI_API_KEY` | Yes, for `batch` mode | *(none)* | **Secret.** API key for the Gemini provider. `batch` mode fails to start without it. |
| `LITELLM_MASTER_KEY` | Yes, for `dependency` mode | *(none)* | **Secret.** Master key for the LiteLLM proxy. `dependency` mode fails to start without it. |
| `LITELLM_BASE_URL` | No | `http://127.0.0.1:9177/v1` | Base URL of the LiteLLM proxy used by `dependency` mode. |
| `LOG_LEVEL` | No | `info` | Logging verbosity. Accepts `error`, `warn`, `info`, `debug`. Any other value (or unset) falls back to `info`. |

## When each is needed

- **`batch` mode** (`llm-site-reviewer batch`) calls `review.LiteLLMRequest`, which builds a
  Gemini client. It reads **`GEMINI_API_KEY`** and errors with `GEMINI_API_KEY not set` if it is empty.
- **`dependency` mode** (`llm-site-reviewer dependency`) builds a LiteLLM proxy client. It reads
  **`LITELLM_MASTER_KEY`** (errors with `LITELLM_MASTER_KEY not set` if empty) and
  **`LITELLM_BASE_URL`** for the proxy address, defaulting to `http://127.0.0.1:9177/v1`.
- **`LOG_LEVEL`** is read once at startup by `logging.InitFromEnv` and sets the log level for
  both modes.

## Secrets

`GEMINI_API_KEY` and `LITELLM_MASTER_KEY` are credentials. Keep them out of source control and
pass them through the environment or a secret store, not command-line flags.
