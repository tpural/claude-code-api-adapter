# claude-code-api-adapter

OpenAI-compatible API server for the Claude Code CLI.

## Why

I built this adapter service in order to connect Claude Code to n8n and other tools in my homelab without paying for API credits. I can leverage my existing Claude Code subscription and pay a fixed monthly cost. 

## Roadmap

| Feature | Description | Status |
|---|---|---|
| Usage tracking | Expose Claude Code usage and limits via the API | Planned |
| Skills management | Upload and reference skill files from chat requests | Planned |

## Quick Start

```bash
go build -o bin/claude-code-api-adapter ./cmd/claude-code-api-adapter
./bin/claude-code-api-adapter
```

Or with Docker:

```bash
docker build -t claude-code-api-adapter .
docker run -p 8080:8080 -e CLAUDE_CODE_OAUTH_TOKEN=your-token claude-code-api-adapter
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Listen port |
| `SESSIONS_DIR` | `./sessions` | Session storage directory |
| `CLAUDE_CODE_OAUTH_TOKEN` | -- | Claude Code auth token |

## Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/v1/chat/completions` | Chat completions (streaming and non-streaming) |
| GET | `/v1/models` | List available models |
| GET | `/health` | Health check |

## Testing

```bash
go test ./...
```

## License

See [LICENSE](LICENSE) for details.
