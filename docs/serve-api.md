# Serve API

`grapher serve` starts an HTTP server exposing analysis as a REST API. The full spec is in `openapi.yaml`.

## Start the server

```bash
grapher serve --repo ./my-repo --port 8080
```

## Endpoints

### POST /api/v1/analyze

Trigger a new analysis job. Returns a job ID immediately; analysis runs asynchronously.

```bash
curl -X POST http://localhost:8080/api/v1/analyze \
  -H "Content-Type: application/json" \
  -d '{"repo": "./my-repo", "analyzers": ["deadcode"]}'
# → {"id": "550e8400-e29b-41d4-a716-446655440000"}
```

Only one job runs at a time — returns `409` if a job is already running.

### GET /api/v1/jobs/{id}

Poll job status. Status values: `pending`, `running`, `done`, `error`.

```bash
curl http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000
# → {"id": "...", "status": "done", "findings": [...]}
```

### GET /api/v1/findings

List findings from the most recent completed job. Supports filtering and sorting.

```bash
# Filter by analyzer
curl "http://localhost:8080/api/v1/findings?analyzer=deadcode"

# Sort by centrality (descending)
curl "http://localhost:8080/api/v1/findings?sort=centrality"
```

### GET /api/v1/graph

Returns the full code graph as JSON — nodes and edges — suitable for visualization.

```bash
curl http://localhost:8080/api/v1/graph
# → {"nodes": [...], "edges": [...]}
```

### POST /api/v1/fix

Request a Claude fix proposal for a specific finding by its index in the findings array.

```bash
curl -X POST http://localhost:8080/api/v1/fix \
  -H "Content-Type: application/json" \
  -d '{"finding_index": 0}'
# → {"proposal": "..."}
```

Requires `ANTHROPIC_API_KEY` environment variable to be set on the server.

## Notes

- Jobs are stored in memory — they are lost on server restart
- Only one concurrent job is supported in v1
- See `openapi.yaml` for full request/response schemas
