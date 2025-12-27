"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  allocateNode,
  ApiError,
  completeNode,
  createNode,
  listNodes,
  listResources,
  moveNode,
} from "../lib/api";
import type { Node, Resource } from "../lib/types";
import { ApiLog, type ApiLogEntry } from "./ApiLog";
import { CreateNodeForm } from "./CreateNodeForm";
import { NodeCard } from "./NodeCard";
import { ResourceCard } from "./ResourceCard";
import { Toast } from "./Toast";

function nowTime(): string {
  return new Date().toLocaleTimeString();
}

function getApiBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
  return base.replace(/\/+$/, "");
}


export function QueueManager() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);
  const [apiLogEntries, setApiLogEntries] = useState<ApiLogEntry[]>([]);

  const addApiLog = useCallback((entry: Omit<ApiLogEntry, "time">) => {
    setApiLogEntries((prev) => {
      const next = [...prev, { time: nowTime(), ...entry }];
      return next.length > 100 ? next.slice(next.length - 100) : next;
    });
  }, []);

  const refresh = useCallback(
    async (opts?: { log?: boolean }) => {
      const logCalls = opts?.log ?? true;
      try {
        const [r, n] = await Promise.all([listResources(), listNodes()]);
        if (logCalls) {
          addApiLog({ method: "GET", url: "/resources", status: 200 });
          addApiLog({ method: "GET", url: "/nodes", status: 200 });
        }
        setResources(r);
        setNodes(n);
      } catch (e) {
        const err = e as Error;
        // If this was a logged call, record failure; for polling errors keep quiet.
        if (logCalls) {
          addApiLog({
            method: "GET",
            url: "/resources|/nodes",
            status: e instanceof ApiError ? e.status : 0,
            error: err.message,
          });
          setToast({ kind: "error", message: err.message });
        }
        throw e;
      }
    },
    [addApiLog],
  );

  useEffect(() => {
    (async () => {
      try {
        await refresh({ log: true });
      } finally {
        setLoading(false);
      }
    })();
  }, [refresh]);

  // Poll every 2 seconds; do not spam the API log (only surface interactive calls).
  useEffect(() => {
    const t = setInterval(() => {
      refresh({ log: false }).catch(() => {});
    }, 2000);
    return () => clearInterval(t);
  }, [refresh]);

  const unassignedNodes = useMemo(
    () => nodes.filter((n) => !n.resource_id && !n.completed),
    [nodes],
  );

  const onCreate = useCallback(
    async ({ entityName, resourceId }: { entityName: string; resourceId?: string }) => {
      try {
        const body = { entity_name: entityName, ...(resourceId ? { resource_id: resourceId } : {}) };
        const node = await createNode(entityName, resourceId);
        addApiLog({ method: "POST", url: "/nodes", status: 201, body });
        setToast({ kind: "success", message: `Node created: ${node.id}` });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        addApiLog({
          method: "POST",
          url: "/nodes",
          status: e instanceof ApiError ? e.status : 0,
          body: { entity_name: entityName, ...(resourceId ? { resource_id: resourceId } : {}) },
          error: err.message,
        });
        setToast({ kind: "error", message: err.message });
      }
    },
    [addApiLog, refresh],
  );

  const onMove = useCallback(
    async (nodeId: string) => {
      const targetResourceId = window.prompt("Enter target resource ID:");
      if (!targetResourceId) return;
      try {
        const body = { target_resource_id: targetResourceId };
        await moveNode(nodeId, targetResourceId);
        addApiLog({ method: "POST", url: `/nodes/${nodeId}/move`, status: 200, body });
        setToast({ kind: "success", message: "Node moved successfully" });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        addApiLog({
          method: "POST",
          url: `/nodes/${nodeId}/move`,
          status: e instanceof ApiError ? e.status : 0,
          body: { target_resource_id: targetResourceId },
          error: err.message,
        });
        setToast({ kind: "error", message: err.message });
      }
    },
    [addApiLog, refresh],
  );

  const onAllocate = useCallback(
    async (nodeId: string) => {
      try {
        await allocateNode(nodeId);
        addApiLog({ method: "POST", url: `/nodes/${nodeId}/allocate`, status: 200 });
        setToast({ kind: "success", message: "Node allocated successfully" });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        addApiLog({
          method: "POST",
          url: `/nodes/${nodeId}/allocate`,
          status: e instanceof ApiError ? e.status : 0,
          error: err.message,
        });
        setToast({ kind: "error", message: err.message });
      }
    },
    [addApiLog, refresh],
  );

  const onComplete = useCallback(
    async (nodeId: string) => {
      const ok = window.confirm("Are you sure you want to complete this node?");
      if (!ok) return;
      try {
        await completeNode(nodeId);
        addApiLog({ method: "POST", url: `/nodes/${nodeId}/complete`, status: 200 });
        setToast({ kind: "success", message: "Node completed successfully" });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        addApiLog({
          method: "POST",
          url: `/nodes/${nodeId}/complete`,
          status: e instanceof ApiError ? e.status : 0,
          error: err.message,
        });
        setToast({ kind: "error", message: err.message });
      }
    },
    [addApiLog, refresh],
  );

  const onAddToResource = useCallback(
    async (nodeId: string) => {
      const resourceId = window.prompt("Enter resource ID:");
      if (!resourceId) return;
      try {
        const body = { target_resource_id: resourceId };
        await moveNode(nodeId, resourceId);
        addApiLog({ method: "POST", url: `/nodes/${nodeId}/move`, status: 200, body });
        setToast({ kind: "success", message: "Node added to resource successfully" });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        addApiLog({
          method: "POST",
          url: `/nodes/${nodeId}/move`,
          status: e instanceof ApiError ? e.status : 0,
          body: { target_resource_id: resourceId },
          error: err.message,
        });
        setToast({ kind: "error", message: err.message });
      }
    },
    [addApiLog, refresh],
  );

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Queue Manager Simulation</h1>
        <p className="mt-2 text-white/80">
          React + Next.js UI for NodeQueue. Server is running at           
          <code className="rounded bg-white/10 px-1.5 py-0.5">
          {getApiBaseUrl()}
          </code>.
        </p>
      </header>

      {toast ? (
        <Toast kind={toast.kind} message={toast.message} onClose={() => setToast(null)} />
      ) : null}

      <CreateNodeForm resources={resources} onCreate={onCreate} disabled={loading} />

      {unassignedNodes.length > 0 ? (
        <section className="mb-6 rounded-xl bg-white p-5 shadow-sm">
          <div className="mb-3 text-lg font-semibold text-zinc-900">Unassigned Nodes</div>
          <div className="flex flex-wrap gap-2">
            {unassignedNodes.map((n) => (
              <NodeCard
                key={n.id}
                node={n}
                context="unassigned"
                onAllocate={onAllocate}
                onMove={onMove}
                onComplete={onComplete}
                onAddToResource={onAddToResource}
              />
            ))}
          </div>
        </section>
      ) : null}

      <section className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
        {resources.map((r) => (
          <ResourceCard
            key={r.id}
            resource={r}
            onAllocate={onAllocate}
            onMove={onMove}
            onComplete={onComplete}
          />
        ))}
      </section>

      <ApiLog entries={apiLogEntries} onClear={() => setApiLogEntries([])} />
    </div>
  );
}


