type JsonRpcResponse =
  | { jsonrpc: "2.0"; id: number; result: { content?: Array<{ type: string; text?: string }> } }
  | { jsonrpc: "2.0"; id?: number; error: { code: number; message: string } };
let nextId = 1;

function getQueueAgentBaseUrl(): string {
  // Server-side env var (Next runtime). In Docker this should be http://queue-agent:8090
  const base = process.env.QUEUE_AGENT_BASE_URL || "http://localhost:8090";
  return base.replace(/\/+$/, "");
}

async function send(method: string, params?: unknown): Promise<JsonRpcResponse> {
  const id = nextId++;
  const msg = { jsonrpc: "2.0" as const, id, method, ...(params ? { params } : {}) };

  const res = await fetch(`${getQueueAgentBaseUrl()}/rpc`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(msg),
    cache: "no-store",
  });
  const json = (await res.json()) as JsonRpcResponse;
  // Even on non-2xx we still expect a JSON-RPC response; surface HTTP failures explicitly.
  if (!res.ok) {
    const msg = "error" in json && json.error?.message ? json.error.message : res.statusText;
    throw new Error(`queue-agent /rpc failed: ${msg}`);
  }
  return json;
}

function extractTexts(resp: JsonRpcResponse): string[] {
  if ("error" in resp && resp.error) return [`[error] ${resp.error.message}`];
  const content = ("result" in resp ? resp.result?.content : undefined) ?? [];
  return content
    .filter((c) => c && typeof c === "object" && c.type === "text")
    .map((c) => (c.text ?? "").trim())
    .filter(Boolean);
}

export async function ensureInitialized(): Promise<void> {
  const initResp = await send("initialize", {
    clientInfo: { name: "queue-ui", version: "0.0.0" },
    protocolVersion: "2024-11-05",
    capabilities: {},
  });
  if ("error" in initResp && initResp.error) {
    throw new Error(`initialize failed: ${initResp.error.message}`);
  }

  // No local state; queue-agent holds cache in its own process.
}

export async function refreshMetrics(): Promise<{ nodes: string[]; resources: string[] }> {
  const nodesResp = await send("tools/call", { name: "queue_nodes_metrics", arguments: {} });
  const resResp = await send("tools/call", { name: "queue_resources_metrics", arguments: {} });
  return { nodes: extractTexts(nodesResp), resources: extractTexts(resResp) };
}

export async function askMetricsQuestion(question: string): Promise<{ answer: string; context?: string; rawParts: string[] }> {
  const resp = await send("tools/call", {
    name: "queue_metrics_question",
    arguments: { question },
  });
  const parts = extractTexts(resp);
  return { answer: parts[0] ?? "", context: parts[1], rawParts: parts };
}

