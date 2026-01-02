"use client";

import React from "react";
import type { Node, NodeMetrics, Resource } from "../lib/types";
import { NodeCard } from "./NodeCard";

type DropKind = "waiting" | "service";

function getDraggedNodeId(e: React.DragEvent): string | null {
  const id =
    e.dataTransfer.getData("application/x-node-id") ||
    e.dataTransfer.getData("text/plain") ||
    "";
  return id.trim() ? id.trim() : null;
}

export function ResourceCard({
  resource,
  onComplete,
  onDropNode,
  nodeMetricsById,
}: {
  resource: Resource;
  onComplete: (nodeId: string) => void;
  onDropNode: (args: { nodeId: string; resourceId: string; kind: DropKind }) => void;
  nodeMetricsById?: Record<string, NodeMetrics | undefined>;
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
        <div
          className="min-h-[80px] rounded-lg border-2 border-dashed border-emerald-500 bg-emerald-50 p-3"
          onDragOver={(e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = "move";
          }}
          onDrop={(e) => {
            e.preventDefault();
            const nodeId = getDraggedNodeId(e);
            if (!nodeId) return;
            onDropNode({ nodeId, resourceId: resource.id, kind: "service" });
          }}
        >
          {serviceNodes.length === 0 ? (
            <div className="py-4 text-center text-sm italic text-zinc-400">
              Drop a node here to allocate it into service
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {serviceNodes.map((n) => (
                <NodeCard
                  key={n.id}
                  node={n}
                  context="service"
                  onComplete={onComplete}
                  metrics={nodeMetricsById?.[n.id]}
                  currentResourceId={resource.id}
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
        <div
          className="min-h-[60px] rounded-lg border-2 border-dashed border-amber-500 bg-amber-50 p-3"
          onDragOver={(e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = "move";
          }}
          onDrop={(e) => {
            e.preventDefault();
            const nodeId = getDraggedNodeId(e);
            if (!nodeId) return;
            onDropNode({ nodeId, resourceId: resource.id, kind: "waiting" });
          }}
        >
          {waitingNodes.length === 0 ? (
            <div className="py-4 text-center text-sm italic text-zinc-400">
              Drop a node here to move it into waiting
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {waitingNodes.map((n) => (
                <NodeCard
                  key={n.id}
                  node={n}
                  context="waiting"
                  onComplete={onComplete}
                  metrics={nodeMetricsById?.[n.id]}
                  currentResourceId={resource.id}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}


