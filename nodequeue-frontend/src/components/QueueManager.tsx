"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  allocateNode,
  ApiError,
  completeNode,
  createNode,
  getNodesMetrics,
  listNodes,
  listResources,
  moveNode,
} from "../lib/api";
import type { Node, NodeMetrics, NodesMetricsResponse, Resource } from "../lib/types";
import { ApiLog, type ApiLogEntry } from "./ApiLog";
import { CreateNodeForm } from "./CreateNodeForm";
import { NodeMetricsFrame } from "./NodeMetricsFrame";
import { NodeCard } from "./NodeCard";
import { ResourceCard } from "./ResourceCard";
import { Toast } from "./Toast";

function nowTime(): string {
  return new Date().toLocaleTimeString();
}

function getApiBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL || "/api";
  return base.replace(/\/+$/, "");
}


export function QueueManager() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);
  const [apiLogEntries, setApiLogEntries] = useState<ApiLogEntry[]>([]);
  const [metrics, setMetrics] = useState<NodesMetricsResponse | null>(null);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [metricsError, setMetricsError] = useState<string | null>(null);
  const [metricsLastUpdated, setMetricsLastUpdated] = useState<string | null>(null);

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

  const refreshMetrics = useCallback(async () => {
    setMetricsLoading(true);
    setMetricsError(null);
    try {
      const data = await getNodesMetrics();
      setMetrics(data);
      setMetricsLastUpdated(new Date().toLocaleTimeString());
    } catch (e) {
      const err = e as Error;
      setMetricsError(err.message);
    } finally {
      setMetricsLoading(false);
    }
  }, []);

  // Poll metrics every 10s.
  useEffect(() => {
    refreshMetrics().catch(() => {});
    const t = setInterval(() => {
      refreshMetrics().catch(() => {});
    }, 10000);
    return () => clearInterval(t);
  }, [refreshMetrics]);

  const nodeMetricsById = useMemo(() => {
    const out: Record<string, NodeMetrics> = {};
    for (const m of metrics?.active_nodes ?? []) out[m.id] = m;
    for (const m of metrics?.completed_nodes ?? []) out[m.id] = m;
    return out;
  }, [metrics]);

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

  const onDropNode = useCallback(
    async (args: { nodeId: string; resourceId: string; kind: "waiting" | "service" }) => {
      const { nodeId, resourceId, kind } = args;
      const current = nodes.find((n) => n.id === nodeId);
      try {
        // 1) Ensure node is assigned to the target resource (Move always enqueues into waiting).
        if (current?.resource_id !== resourceId) {
          const body = { target_resource_id: resourceId };
          await moveNode(nodeId, resourceId);
          addApiLog({ method: "POST", url: `/nodes/${nodeId}/move`, status: 200, body });
        } else if (kind === "waiting") {
          // Dropping into waiting can be used to "de-allocate" within the same resource.
          // MoveNode removes the node from current queues and re-enqueues into waiting.
          const body = { target_resource_id: resourceId };
          await moveNode(nodeId, resourceId);
          addApiLog({ method: "POST", url: `/nodes/${nodeId}/move`, status: 200, body });
        }

        // 2) If dropped onto service, allocate into service (capacity enforced by API).
        if (kind === "service") {
          await allocateNode(nodeId);
          addApiLog({ method: "POST", url: `/nodes/${nodeId}/allocate`, status: 200 });
        }

        setToast({
          kind: "success",
          message:
            kind === "service"
              ? `Node allocated to ${resourceId}`
              : `Node moved to waiting in ${resourceId}`,
        });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        addApiLog({
          method: "DND",
          url: kind === "service" ? `/nodes/${nodeId}/move+allocate` : `/nodes/${nodeId}/move`,
          status: e instanceof ApiError ? e.status : 0,
          body: { resource_id: resourceId, kind },
          error: err.message,
        });
        setToast({ kind: "error", message: err.message });
      }
    },
    [addApiLog, refresh, nodes],
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
                onComplete={onComplete}
                metrics={nodeMetricsById[n.id]}
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
            onComplete={onComplete}
            onDropNode={onDropNode}
            nodeMetricsById={nodeMetricsById}
          />
        ))}
      </section>

      <NodeMetricsFrame
        metrics={metrics}
        loading={metricsLoading}
        error={metricsError}
        lastUpdatedAt={metricsLastUpdated}
      />

      <ApiLog entries={apiLogEntries} onClear={() => setApiLogEntries([])} />
    </div>
  );
}


