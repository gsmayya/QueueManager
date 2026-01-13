import type { AdminEntity, AdminRoom, AdminUser, ErrorResponse } from "./types";
import { ApiError } from "./api";

function getMasterApiBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_MASTER_API_BASE_URL || "/master-api";
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
  const url = joinUrl(getMasterApiBaseUrl(), path);
  const res = await fetch(url, init);
  if (!res.ok) {
    const { message, details } = await parseError(res);
    throw new ApiError(message, res.status, details);
  }
  // Some endpoints might return 204; callers should not use requestJson for those.
  return (await res.json()) as T;
}

async function requestNoContent(path: string, init: RequestInit): Promise<void> {
  const url = joinUrl(getMasterApiBaseUrl(), path);
  const res = await fetch(url, init);
  if (!res.ok) {
    const { message, details } = await parseError(res);
    throw new ApiError(message, res.status, details);
  }
}

// --- Entities

export async function listAdminEntities(): Promise<AdminEntity[]> {
  return requestJson<AdminEntity[]>("/entities", { method: "GET" });
}

export async function listEntitiesByPhone(phone: string): Promise<AdminEntity[]> {
  const p = encodeURIComponent(phone.trim());
  return requestJson<AdminEntity[]>(`/entities?phone=${p}`, { method: "GET" });
}

export async function getEntityById(id: string): Promise<AdminEntity> {
  return requestJson<AdminEntity>(`/entities/${id}`, { method: "GET" });
}

export async function createAdminEntity(args: {
  name: string;
  phone: string;
  email?: string;
}): Promise<AdminEntity> {
  return requestJson<AdminEntity>("/entities", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(args),
  });
}

export async function updateAdminEntity(
  id: string,
  patch: Partial<{ name: string; phone: string; email: string; joining_date: string }>,
): Promise<AdminEntity> {
  return requestJson<AdminEntity>(`/entities/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

export async function deleteAdminEntity(id: string): Promise<void> {
  return requestNoContent(`/entities/${id}`, { method: "DELETE" });
}

// --- Users

export async function listAdminUsers(): Promise<AdminUser[]> {
  return requestJson<AdminUser[]>("/users", { method: "GET" });
}

export async function createAdminUser(args: {
  user_id: string;
  name: string;
  email: string;
  password: string;
}): Promise<AdminUser> {
  return requestJson<AdminUser>("/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(args),
  });
}

export async function updateAdminUser(
  id: string,
  patch: Partial<{ user_id: string; name: string; email: string; password: string }>,
): Promise<AdminUser> {
  return requestJson<AdminUser>(`/users/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

export async function deleteAdminUser(id: string): Promise<void> {
  return requestNoContent(`/users/${id}`, { method: "DELETE" });
}

// --- Rooms

export async function listAdminRooms(opts?: { includeDeleted?: boolean }): Promise<AdminRoom[]> {
  const includeDeleted = opts?.includeDeleted ?? true;
  const qs = `?include_deleted=${includeDeleted ? "true" : "false"}`;
  return requestJson<AdminRoom[]>(`/rooms${qs}`, { method: "GET" });
}

export async function createAdminRoom(args: { name: string; capacity: number; id?: string }): Promise<AdminRoom> {
  return requestJson<AdminRoom>("/rooms", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(args),
  });
}

export async function updateAdminRoom(
  id: string,
  patch: Partial<{ name: string; capacity: number }>,
): Promise<AdminRoom> {
  return requestJson<AdminRoom>(`/rooms/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
}

export async function deleteAdminRoom(id: string): Promise<void> {
  return requestNoContent(`/rooms/${id}`, { method: "DELETE" });
}

