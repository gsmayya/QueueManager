"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import type { AdminEntity, Node, Resource } from "../../lib/types";
import { completeNode, createNodeFromEntity, listNodes, listResources } from "../../lib/api";
import { getEntityById, listEntitiesByPhone } from "../../lib/masterApi";
import { Toast } from "../Toast";

function nowTime(): string {
  return new Date().toLocaleTimeString();
}

function gen4(): string {
  const n = Math.floor(Math.random() * 10000);
  return String(n).padStart(4, "0");
}

async function generateUniqueNodeName(activeNodes: Node[]): Promise<string> {
  const used = new Set<string>();
  for (const n of activeNodes) {
    if (n.node_name && n.node_name.trim()) used.add(n.node_name.trim());
  }
  // 10k space; retry a bit.
  for (let i = 0; i < 20000; i++) {
    const cand = gen4();
    if (!used.has(cand)) return cand;
  }
  throw new Error("Unable to generate a unique 4-digit node_name (too many active nodes).");
}

export function NodePage() {
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);

  const [resources, setResources] = useState<Resource[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<string | null>(null);

  const [phone, setPhone] = useState("");
  const [searchLoading, setSearchLoading] = useState(false);
  const [matches, setMatches] = useState<AdminEntity[]>([]);
  const [selectedEntityId, setSelectedEntityId] = useState<string>("");
  const [initialRoomId, setInitialRoomId] = useState<string>("");

  const [entityCache, setEntityCache] = useState<Record<string, AdminEntity>>({});

  const refresh = useCallback(async () => {
    try {
      const [r, n] = await Promise.all([listResources(), listNodes()]);
      setResources(r);
      setNodes(n);
      setLastUpdated(nowTime());
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh().catch(() => {});
    const t = setInterval(() => refresh().catch(() => {}), 2000);
    return () => clearInterval(t);
  }, [refresh]);

  // Cache customer details for displayed nodes.
  useEffect(() => {
    const ids = new Set<string>();
    for (const n of nodes) {
      const id = n.entity?.id;
      if (id) ids.add(id);
    }
    const missing = [...ids].filter((id) => !entityCache[id]);
    if (missing.length === 0) return;

    (async () => {
      const entries = await Promise.all(
        missing.map(async (id) => {
          try {
            const e = await getEntityById(id);
            return [id, e] as const;
          } catch {
            return null;
          }
        }),
      );
      setEntityCache((prev) => {
        const next = { ...prev };
        for (const it of entries) {
          if (!it) continue;
          next[it[0]] = it[1];
        }
        return next;
      });
    })();
  }, [nodes, entityCache]);

  const activeNodes = useMemo(() => nodes.filter((n) => !n.completed), [nodes]);

  const onSearch = useCallback(async () => {
    const p = phone.trim();
    if (!p) return;
    setSearchLoading(true);
    try {
      const res = await listEntitiesByPhone(p);
      setMatches(res);
      setSelectedEntityId(res.length === 1 ? res[0].id : "");
    } catch (e) {
      setToast({ kind: "error", message: (e as Error).message });
    } finally {
      setSearchLoading(false);
    }
  }, [phone]);

  const onCreate = useCallback(async () => {
    const entityId = selectedEntityId.trim();
    if (!entityId) {
      setToast({ kind: "error", message: "Select an entity first." });
      return;
    }
    try {
      const nodeName = await generateUniqueNodeName(activeNodes);
      await createNodeFromEntity({
        entity_id: entityId,
        node_name: nodeName,
        resource_id: initialRoomId || undefined,
      });
      setToast({ kind: "success", message: `Node created: ${nodeName}` });
      await refresh();
    } catch (e) {
      setToast({ kind: "error", message: (e as Error).message });
    }
  }, [selectedEntityId, activeNodes, initialRoomId, refresh]);

  const onComplete = useCallback(
    async (nodeId: string) => {
      const ok = window.confirm("Complete this node?");
      if (!ok) return;
      try {
        await completeNode(nodeId);
        setToast({ kind: "success", message: "Node completed" });
        await refresh();
      } catch (e) {
        const err = e as Error;
        setToast({ kind: "error", message: err.message });
      }
    },
    [refresh],
  );

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Nodes</h1>
        <p className="mt-2 text-white/80">Create nodes by looking up customers via mobile number.</p>
      </header>

      {toast ? <Toast kind={toast.kind} message={toast.message} onClose={() => setToast(null)} /> : null}

      <section className="mb-6 rounded-xl bg-white p-5 shadow-sm">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div className="text-lg font-semibold text-zinc-900">Create Node</div>
          <div className="text-xs text-zinc-500">
            {loading ? "Loading…" : "Ready"}
            {lastUpdated ? <span className="ml-2">Last updated: {lastUpdated}</span> : null}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
          <div className="md:col-span-1">
            <div className="text-xs font-semibold tracking-widest text-zinc-500">MOBILE NUMBER</div>
            <div className="mt-2 flex gap-2">
              <input
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="e.g. 5551234"
                className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-indigo-500 focus:outline-none"
              />
              <button
                className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-semibold text-white hover:bg-zinc-800 disabled:bg-zinc-300"
                onClick={onSearch}
                disabled={searchLoading}
              >
                Search
              </button>
            </div>
            <div className="mt-2 text-xs text-zinc-500">Calls master-service: /entities?phone=…</div>
          </div>

          <div className="md:col-span-1">
            <div className="text-xs font-semibold tracking-widest text-zinc-500">SELECT CUSTOMER</div>
            <select
              value={selectedEntityId}
              onChange={(e) => setSelectedEntityId(e.target.value)}
              className="mt-2 w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
            >
              <option value="">{matches.length === 0 ? "Search by phone first" : "Select a match"}</option>
              {matches.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name} — {m.phone} {m.email ? `(${m.email})` : ""}
                </option>
              ))}
            </select>
            {matches.length > 1 ? (
              <div className="mt-2 text-xs text-amber-700">Multiple matches found. Pick one.</div>
            ) : null}
            {matches.length === 0 && phone.trim() ? (
              <div className="mt-2 text-xs text-zinc-500">No matches yet (try Search).</div>
            ) : null}
          </div>

          <div className="md:col-span-1">
            <div className="text-xs font-semibold tracking-widest text-zinc-500">OPTIONAL INITIAL ROOM</div>
            <select
              value={initialRoomId}
              onChange={(e) => setInitialRoomId(e.target.value)}
              className="mt-2 w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
            >
              <option value="">None</option>
              {resources.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name || r.id} (cap {r.capacity})
                </option>
              ))}
            </select>
            <button
              className="mt-3 w-full rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:bg-zinc-300"
              onClick={onCreate}
              disabled={!selectedEntityId || loading}
            >
              Create Node (generates 4-digit node_name)
            </button>
          </div>
        </div>
      </section>

      <section className="rounded-xl bg-white p-5 shadow-sm">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div className="text-lg font-semibold text-zinc-900">Nodes</div>
          <div className="flex flex-wrap gap-2 text-sm">
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Active: <span className="font-semibold">{activeNodes.length}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Completed: <span className="font-semibold">{nodes.length - activeNodes.length}</span>
            </div>
          </div>
        </div>

        <div className="overflow-auto rounded-lg border border-zinc-200">
          <table className="min-w-[1100px] w-full text-left text-sm">
            <thead className="bg-zinc-50 text-xs font-semibold tracking-widest text-zinc-500">
              <tr>
                <th className="px-4 py-3">NODE</th>
                <th className="px-4 py-3">CUSTOMER</th>
                <th className="px-4 py-3">PHONE</th>
                <th className="px-4 py-3">EMAIL</th>
                <th className="px-4 py-3">ROOM</th>
                <th className="px-4 py-3">STATE</th>
                <th className="px-4 py-3">ACTIONS</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {nodes.map((n) => {
                const entId = n.entity?.id;
                const ent = entId ? entityCache[entId] : undefined;
                return (
                  <tr key={n.id} className="hover:bg-zinc-50">
                    <td className="px-4 py-3 font-mono text-zinc-900">{n.node_name || "—"}</td>
                    <td className="px-4 py-3 font-semibold text-zinc-900">{ent?.name || n.entity?.name || "—"}</td>
                    <td className="px-4 py-3 text-zinc-700">{ent?.phone || "—"}</td>
                    <td className="px-4 py-3 text-zinc-700">{ent?.email || "—"}</td>
                    <td className="px-4 py-3 text-zinc-700">{n.resource_id || "—"}</td>
                    <td className="px-4 py-3 text-zinc-700">{n.completed ? "completed" : "active"}</td>
                    <td className="px-4 py-3">
                      {!n.completed ? (
                        <button
                          className="rounded bg-rose-600 px-3 py-1 text-xs font-semibold text-white hover:bg-rose-500"
                          onClick={() => onComplete(n.id)}
                        >
                          Complete
                        </button>
                      ) : (
                        <span className="text-xs text-zinc-500">—</span>
                      )}
                    </td>
                  </tr>
                );
              })}
              {nodes.length === 0 ? (
                <tr>
                  <td className="px-4 py-6 text-sm text-zinc-600" colSpan={7}>
                    No nodes yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

