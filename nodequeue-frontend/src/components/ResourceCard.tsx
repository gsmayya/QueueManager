"use client";

import React from "react";
import type { Node, Resource } from "../lib/types";
import { NodeCard } from "./NodeCard";

export function ResourceCard({
  resource,
  onAllocate,
  onMove,
  onComplete,
}: {
  resource: Resource;
  onAllocate: (nodeId: string) => void;
  onMove: (nodeId: string) => void;
  onComplete: (nodeId: string) => void;
}) {
  const serviceNodes: Node[] = resource.nodes ?? [];
  const waitingNodes: Node[] = resource.waiting_queue ?? [];

  return (
    <div className="rounded-xl bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
      <div className="mb-4 flex items-center justify-between border-b border-zinc-100 pb-4">
        <div className="text-lg font-semibold text-zinc-900">{resource.id}</div>
        <div className="rounded-full bg-indigo-600 px-3 py-1 text-sm font-semibold text-white">
          {serviceNodes.length} / {resource.capacity}
        </div>
      </div>

      <div className="mb-5">
        <div className="mb-2 text-xs font-semibold tracking-widest text-zinc-500">
          SERVICE QUEUE ({serviceNodes.length})
        </div>
        <div className="min-h-[80px] rounded-lg border-2 border-dashed border-emerald-500 bg-emerald-50 p-3">
          {serviceNodes.length === 0 ? (
            <div className="py-4 text-center text-sm italic text-zinc-400">
              Empty
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {serviceNodes.map((n) => (
                <NodeCard
                  key={n.id}
                  node={n}
                  context="service"
                  onAllocate={onAllocate}
                  onMove={onMove}
                  onComplete={onComplete}
                  onAddToResource={onMove}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      <div>
        <div className="mb-2 text-xs font-semibold tracking-widest text-zinc-500">
          WAITING QUEUE ({waitingNodes.length})
        </div>
        <div className="min-h-[60px] rounded-lg border-2 border-dashed border-amber-500 bg-amber-50 p-3">
          {waitingNodes.length === 0 ? (
            <div className="py-4 text-center text-sm italic text-zinc-400">
              Empty
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {waitingNodes.map((n) => (
                <NodeCard
                  key={n.id}
                  node={n}
                  context="waiting"
                  onAllocate={onAllocate}
                  onMove={onMove}
                  onComplete={onComplete}
                  onAddToResource={onMove}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}


