# queue-agent (MCP server)

Minimal, dependency-free MCP-style server over **HTTP JSON-RPC** for **queue-service metrics**.

## Metrics tools (connect to queue-service)

Set `QUEUE_SERVICE_BASE_URL` (default is `http://localhost:8080`) and run the server (HTTP):

```bash
cd queue-agent
QUEUE_AGENT_HTTP_ADDR=:8090 QUEUE_SERVICE_BASE_URL=http://localhost:8080 go run .
```

Then POST JSON-RPC to:
- `POST /rpc` (JSON-RPC request/response)
- `GET /healthz`

## Tools

The server supports:
- `initialize`
- `tools/list`
- `tools/call` for:
  - `queue_nodes_metrics`
  - `queue_resources_metrics`
  - `queue_metrics_question`

## Example workflow

Typical flow is:
- fetches `/nodes/metrics`
- fetches `/resources/metrics`
- asks a follow-up question using cached context

