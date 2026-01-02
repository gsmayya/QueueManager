"use client";

import React, { useMemo } from "react";
import type { Node, NodeMetrics } from "../lib/types";

export type NodeContext = "unassigned" | "waiting" | "service";

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}...` : id;
}

function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "0s";
  const s = Math.floor(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = s % 60;
  if (h > 0) return `${h}h ${m}m ${ss}s`;
  if (m > 0) return `${m}m ${ss}s`;
  return `${ss}s`;
}

function getResourceHistory(node: Node): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const entry of node.log ?? []) {
    const rid = (entry.resource_id || "").trim();
    if (!rid) continue;
    if (seen.has(rid)) continue;
    seen.add(rid);
    out.push(rid);
  }
  return out;
}

export function NodeCard({
  node,
  context,
  onComplete,
  draggable,
  metrics,
  currentResourceId,
}: {
  node: Node;
  context: NodeContext;
  onComplete: (nodeId: string) => void;
  draggable?: boolean;
  metrics?: NodeMetrics;
  currentResourceId?: string;
}) {
  const disabled = node.completed;
  const history = useMemo(() => getResourceHistory(node), [node]);

  const waitingInCurrentMS = useMemo(() => {
    if (context !== "waiting") return null;
    if (!metrics || !currentResourceId) return null;
    const segs = metrics.waiting_segments || [];
    for (let i = segs.length - 1; i >= 0; i--) {
      const seg = segs[i];
      if (seg.resource_id === currentResourceId) return seg.duration_ms ?? 0;
    }
    return 0;
  }, [context, metrics, currentResourceId]);

  const waitingByResource = useMemo(() => {
    if (!metrics) return [];
    const map = new Map<string, number>();
    for (const seg of metrics.waiting_segments || []) {
      map.set(seg.resource_id, (map.get(seg.resource_id) ?? 0) + (seg.duration_ms ?? 0));
    }
    return [...map.entries()].sort((a, b) => b[1] - a[1]);
  }, [metrics]);

  return (
    <div
      draggable={!disabled && (draggable ?? true)}
      onDragStart={(e) => {
        // Use a custom MIME type but also set text/plain for compatibility.
        e.dataTransfer.setData("application/x-node-id", node.id);
        e.dataTransfer.setData("text/plain", node.id);
        e.dataTransfer.effectAllowed = "move";
      }}
      className={`rounded-md bg-gradient-to-br from-indigo-500 to-purple-700 px-3 py-2 text-white shadow-sm ${
        disabled ? "opacity-70 grayscale" : ""
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold">{node.entity?.name ?? "node"}</div>
          <div className="text-xs text-white/80">{shortId(node.id)}</div>
        </div>
      </div>

      <div className="mt-2 space-y-1 text-xs text-white/90">
        <div className="text-white/80">
          <span className="font-semibold text-white/90">State:</span>{" "}
          {node.completed ? "completed" : context}
          {!node.completed ? <span className="ml-2 text-white/70">(drag me)</span> : null}
        </div>
        {waitingInCurrentMS != null ? (
          <div className="text-white/80">
            <span className="font-semibold text-white/90">Waiting:</span>{" "}
            {formatDuration(waitingInCurrentMS)}
          </div>
        ) : null}
        <div className="text-white/80">
          <span className="font-semibold text-white/90">Visited:</span>{" "}
          {history.length > 0 ? history.join(" → ") : "—"}
        </div>
        {node.completed && metrics ? (
          <div className="text-white/80">
            <span className="font-semibold text-white/90">Total:</span>{" "}
            {formatDuration(metrics.total_time_in_system_ms)}
          </div>
        ) : null}
      </div>

      <div className="mt-2 flex flex-wrap gap-2">
        {!node.completed ? (
          <>
            <button
              type="button"
              className="rounded bg-rose-600 px-2 py-1 text-xs font-semibold hover:bg-rose-500"
              onClick={() => onComplete(node.id)}
            >
              Complete
            </button>
          </>
        ) : (
          <>
            <span className="text-xs text-white/80">Completed</span>
            {metrics && waitingByResource.length > 0 ? (
              <div className="mt-2 flex w-full flex-wrap gap-2">
                {waitingByResource.map(([rid, ms]) => (
                  <span
                    key={rid}
                    className="rounded-full bg-white/15 px-2 py-1 text-[11px] text-white/90"
                    title={`${rid}: ${ms}ms`}
                  >
                    {rid}: {formatDuration(ms)}
                  </span>
                ))}
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}


