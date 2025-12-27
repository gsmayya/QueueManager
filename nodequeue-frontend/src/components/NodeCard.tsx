"use client";

import React from "react";
import type { Node } from "../lib/types";

export type NodeContext = "unassigned" | "waiting" | "service";

function shortId(id: string): string {
  return id.length > 8 ? `${id.slice(0, 8)}...` : id;
}

export function NodeCard({
  node,
  context,
  onAllocate,
  onMove,
  onComplete,
  onAddToResource,
}: {
  node: Node;
  context: NodeContext;
  onAllocate: (nodeId: string) => void;
  onMove: (nodeId: string) => void;
  onComplete: (nodeId: string) => void;
  onAddToResource: (nodeId: string) => void;
}) {
  const disabled = node.completed;

  return (
    <div
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

      <div className="mt-2 flex flex-wrap gap-2">
        {!node.completed ? (
          <>
            {context === "waiting" ? (
              <button
                type="button"
                className="rounded bg-emerald-600 px-2 py-1 text-xs font-semibold hover:bg-emerald-500"
                onClick={() => onAllocate(node.id)}
              >
                Allocate
              </button>
            ) : null}

            {context === "unassigned" ? (
              <button
                type="button"
                className="rounded bg-amber-500 px-2 py-1 text-xs font-semibold hover:bg-amber-400"
                onClick={() => onAddToResource(node.id)}
              >
                Add to Resource
              </button>
            ) : (
              <button
                type="button"
                className="rounded bg-amber-500 px-2 py-1 text-xs font-semibold hover:bg-amber-400"
                onClick={() => onMove(node.id)}
              >
                Move
              </button>
            )}

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


