"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  allocateNode,
  completeNode,
  listNodes,
  listResources,
  moveNode,
} from "../lib/api";
import type { Node, Resource } from "../lib/types";
import { NodeCard } from "./NodeCard";
import { ResourceCard } from "./ResourceCard";
import { Toast } from "./Toast";

export function QueueManager() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);

  const refresh = useCallback(
    async (opts?: { log?: boolean }) => {
      const logCalls = opts?.log ?? true;
      try {
        const [r, n] = await Promise.all([listResources(), listNodes()]);
        setResources(r);
        setNodes(n);
      } catch (e) {
        const err = e as Error;
        // For polling errors keep quiet; for interactive refresh surface toast.
        if (logCalls) setToast({ kind: "error", message: err.message });
        throw e;
      }
    },
    [],
  );

  useEffect(() => {
    (async () => {
      await refresh({ log: true });
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

  const onComplete = useCallback(
    async (nodeId: string) => {
      const ok = window.confirm("Are you sure you want to complete this node?");
      if (!ok) return;
      try {
        await completeNode(nodeId);
        setToast({ kind: "success", message: "Node completed successfully" });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        setToast({ kind: "error", message: err.message });
      }
    },
    [refresh],
  );

  const onDropNode = useCallback(
    async (args: { nodeId: string; resourceId: string; kind: "waiting" | "service" }) => {
      const { nodeId, resourceId, kind } = args;
      const current = nodes.find((n) => n.id === nodeId);
      try {
        // 1) Ensure node is assigned to the target resource (Move always enqueues into waiting).
        if (current?.resource_id !== resourceId) {
          await moveNode(nodeId, resourceId);
        } else if (kind === "waiting") {
          // Dropping into waiting can be used to "de-allocate" within the same resource.
          // MoveNode removes the node from current queues and re-enqueues into waiting.
          await moveNode(nodeId, resourceId);
        }

        // 2) If dropped onto service, allocate into service (capacity enforced by API).
        if (kind === "service") {
          await allocateNode(nodeId);
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
        setToast({ kind: "error", message: err.message });
      }
    },
    [refresh, nodes],
  );

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Queue Manager UI</h1>        
        <nav className="mt-4 flex justify-center gap-3 text-sm">
          <Link
            href="/node"
            className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15"
          >
            Nodes
          </Link>
          <Link
            href="/rooms/waiting"
            className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15"
          >
            Waiting Rooms
          </Link>
          <Link
            href="/metrics"
            className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15"
          >
            Metrics
          </Link>
          <Link
            href="/admin"
            className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15"
          >
            Admin
          </Link>
        </nav>
      </header>

      {toast ? (
        <Toast kind={toast.kind} message={toast.message} onClose={() => setToast(null)} />
      ) : null}

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
          />
        ))}
      </section>
    </div>
  );
}


