# Personal assistant

Personal assistant is a local Go service that accepts chat requests, calls OpenRouter, and executes Jira actions via tool calls.

[Русская версия](README.ru.md)

## Features

- `POST /chat` endpoint for user messages
- Tool routing from LLM output
- Jira tool execution:
  - `create_project`
  - `search_projects`
  - `delete_project`
- In-memory conversation history
- `GET /memory` endpoint to inspect current history

## How It Works

1. Client sends a JSON request to `POST /chat`.
2. Service builds context from:
   - system prompt
   - chat history
   - new user message
3. OpenRouter returns either plain text or `tool_calls`.
4. If a tool call is returned, the service:
   - parses tool arguments
   - calls Jira API
   - sends tool result back to the model (`role=tool`) to get a final response
5. Service returns model response as JSON.

## Project Structure

- `cmd/main.go` - entrypoint
- `internal/config` - `settings.json` parsing
- `internal/api` - HTTP routes and handlers
- `internal/ai` - OpenRouter requests, tool routing, memory
- `internal/jira` - Jira API wrapper
- `internal/logg` - custom logger (`slog` + colored levels)
- `internal/models` - DTOs split by domain:
  - `internal/models/chat`
  - `internal/models/tool`
  - `internal/models/openrouter`
  - `internal/models/jira`

## Configuration (`settings.json`)

### Core fields

- `api_key_openrouter` - OpenRouter API key. Create it here: <https://openrouter.ai/settings/keys>
- `model_chat_openrouter` - list of chat models. The first one is primary, the rest are fallback options.
- `model_embending_openrouter` - embedding model id for future long-memory/embedding flow.
- `api_url_openrouter` - OpenRouter chat completions URL (usually `https://openrouter.ai/api/v1/chat/completions`).

### Jira fields

- `jira_api_key` - Atlassian API token. Create it here: <https://id.atlassian.com/manage-profile/security/api-tokens>
- `jira_email` - Atlassian account email.
- `jira_personal_url` - Jira base URL in format `https://<your-org>.atlassian.net`.

### Service fields

- `api_host` - HTTP server host (usually `localhost`).
- `api_port` - HTTP server port (usually `8080`).

### Prompt fields

- `promt_system_chat` - main system prompt for assistant behavior.
- `promt_memory_summary` - system prompt used for conversation summarization.
- `memory_summary_user_promt` - user message used to trigger summary generation.

### Context control fields

- `max_tokens_context` - hard limit for context size.
  - `0` means auto mode: service computes limit from selected models.
- `high_border_max_context` - safety margin from maximum context window.
  - example: if model context is `32000` and border is `5000`, effective limit is `27000`.
- `summary_memory_step` - threshold for memory summarization step.
- `division_coefficient` - divides available context into working areas (used by memory-limit logic).

### Security note

`settings.json` is plain text. Do not commit real keys/tokens. Prefer environment variables or a local ignored config file.

## Run

```bash
go run ./cmd
```

Service starts at `http://<api_host>:<api_port>`.

## API

### `POST /chat`

Request body:

```json
{
  "message": "create project TEST in Jira"
}
```

Example:

```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"show Jira projects\"}"
```

Responses:

- `200` - OpenRouter response JSON
- `400` - invalid request body
- `500` - internal processing error (AI/Jira/tool pipeline)

### `GET /memory`

Returns current in-memory chat history.

```bash
curl http://localhost:8080/memory
```

## Logging

Custom logger levels:

- `QUESTION` - incoming user question
- `TASK` - intermediate tool steps
- `ANSWER` - final model answer
- `ERROR` - processing/integration errors
- `INFO` - service lifecycle and state logs

## Limitations

- Memory is process-local (RAM only)
- No persistent storage for chat history
- No built-in authentication for HTTP endpoints
- Long-memory/embeddings flow is not fully finished yet
