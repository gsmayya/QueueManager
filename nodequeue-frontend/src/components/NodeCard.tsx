"use client";

import React, { useMemo } from "react";
import type { Node } from "../lib/types";

export type NodeContext = "unassigned" | "waiting" | "service";

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}...` : id;
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
}: {
  node: Node;
  context: NodeContext;
  onComplete: (nodeId: string) => void;
  draggable?: boolean;
}) {
  const disabled = node.completed;
  const history = useMemo(() => getResourceHistory(node), [node]);

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
        <div className="text-white/80">
          <span className="font-semibold text-white/90">Visited:</span>{" "}
          {history.length > 0 ? history.join(" → ") : "—"}
        </div>
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
          <span className="text-xs text-white/80">Completed</span>
        )}
      </div>
    </div>
  );
}


