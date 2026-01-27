import type {
  ErrorResponse,
  Node,
  NodesMetricsResponse,
  Resource,
  ResourcesSessionMetricsResponse,
  Schedule,
  SchedulesMetricsResponse,
} from "./types";

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

export async function getResourcesMetrics(): Promise<ResourcesSessionMetricsResponse> {
  return requestJson<ResourcesSessionMetricsResponse>("/resources/metrics", { method: "GET" });
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

export async function createNodeFromEntity(args: {
  entity_id: string;
  node_name: string;
  resource_id?: string;
}): Promise<Node> {
  // Keep legacy field entity_name for backward compatibility; backend ignores it when entity_id is provided.
  return requestJson<Node>("/nodes", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      entity_id: args.entity_id,
      node_name: args.node_name,
      entity_name: args.node_name,
      ...(args.resource_id ? { resource_id: args.resource_id } : {}),
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

// --- Schedules

export async function listSchedules(): Promise<Schedule[]> {
  return requestJson<Schedule[]>("/schedules", { method: "GET" });
}

export async function createSchedule(args: {
  entity_id: string;
  resource_id: string;
  interval_seconds: number;
  time_limit_seconds: number;
  waiting_expiry_seconds: number;
  ends_at?: string;
  enabled?: boolean;
  next_run_at?: string;
}): Promise<Schedule> {
  return requestJson<Schedule>("/schedules", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(args),
  });
}

export async function updateSchedule(
  scheduleId: string,
  patch: Partial<{
    resource_id: string;
    interval_seconds: number;
    time_limit_seconds: number;
    waiting_expiry_seconds: number;
    ends_at: string;
    enabled: boolean;
    next_run_at: string;
  }>,
): Promise<Schedule> {
  return requestJson<Schedule>(`/schedules/${scheduleId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

export async function getSchedulesMetrics(): Promise<SchedulesMetricsResponse> {
  return requestJson<SchedulesMetricsResponse>("/schedules/metrics", { method: "GET" });
}


