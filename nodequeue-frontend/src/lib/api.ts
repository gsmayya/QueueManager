import type { ErrorResponse, Node, NodesMetricsResponse, Resource } from "./types";

export class ApiError extends Error {
  status: number;
  details?: unknown;

  constructor(message: string, status: number, details?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.details = details;
  }
}

function getApiBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL || "/api";
  return base.replace(/\/+$/, "");
}

function joinUrl(base: string, path: string): string {
  if (!path.startsWith("/")) return `${base}/${path}`;
  return `${base}${path}`;
}

async function parseError(res: Response): Promise<{ message: string; details?: unknown }> {
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) {
    try {
      const json = (await res.json()) as Partial<ErrorResponse> & Record<string, unknown>;
      return { message: json.error || res.statusText || "Request failed", details: json };
    } catch {
      // fall through
    }
  }
  try {
    const text = await res.text();
    return { message: text || res.statusText || "Request failed", details: text };
  } catch {
    return { message: res.statusText || "Request failed" };
  }
}

async function requestJson<T>(path: string, init: RequestInit): Promise<T> {
  const url = joinUrl(getApiBaseUrl(), path);
  const res = await fetch(url, init);
  if (!res.ok) {
    const { message, details } = await parseError(res);
    throw new ApiError(message, res.status, details);
  }
  return (await res.json()) as T;
}

export async function listResources(): Promise<Resource[]> {
  return requestJson<Resource[]>("/resources", { method: "GET" });
}

export async function listNodes(): Promise<Node[]> {
  return requestJson<Node[]>("/nodes", { method: "GET" });
}

export async function getNodesMetrics(): Promise<NodesMetricsResponse> {
  return requestJson<NodesMetricsResponse>("/nodes/metrics", { method: "GET" });
}

export async function createNode(entityName: string, resourceId?: string): Promise<Node> {
  return requestJson<Node>("/nodes", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      entity_name: entityName,
      ...(resourceId ? { resource_id: resourceId } : {}),
    }),
  });
}

export async function moveNode(nodeId: string, targetResourceId: string): Promise<Node> {
  return requestJson<Node>(`/nodes/${nodeId}/move`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target_resource_id: targetResourceId }),
  });
}

export async function allocateNode(nodeId: string): Promise<Node> {
  return requestJson<Node>(`/nodes/${nodeId}/allocate`, { method: "POST" });
}

export async function completeNode(nodeId: string): Promise<Node> {
  return requestJson<Node>(`/nodes/${nodeId}/complete`, { method: "POST" });
}


