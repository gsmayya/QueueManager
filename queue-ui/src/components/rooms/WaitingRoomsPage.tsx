"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import type { Node, Resource } from "../../lib/types";
import { listResources } from "../../lib/api";
import { Toast } from "../Toast";

function nowTime(): string {
  return new Date().toLocaleTimeString();
}

function displayNodeName(n: Node): string {
  const nn = (n.node_name || "").trim();
  if (nn) return nn;
  const id = (n.id || "").trim();
  return id.length > 8 ? id.slice(0, 8) : id || "—";
}

function compareByNodeNameAsc(a: Node, b: Node): number {
  const aName = (a.node_name || "").trim();
  const bName = (b.node_name || "").trim();
  const aHas = aName.length > 0;
  const bHas = bName.length > 0;
  if (aHas && bHas) return aName.localeCompare(bName);
  if (aHas !== bHas) return aHas ? -1 : 1; // missing node_name goes last
  return displayNodeName(a).localeCompare(displayNodeName(b));
}

function NodeNamePill({ value }: { value: string }) {
  return (
    <span className="rounded-md bg-zinc-900 px-2 py-1 font-mono text-xs font-semibold text-white">
      {value}
    </span>
  );
}

function RoomFrame({ resource }: { resource: Resource }) {
  const serviceNodes = useMemo(() => {
    const ns = (resource.nodes ?? []).slice();
    ns.sort(compareByNodeNameAsc);
    return ns;
  }, [resource.nodes]);

  const waitingNodes = useMemo(() => {
    const ns = (resource.waiting_queue ?? []).slice();
    ns.sort(compareByNodeNameAsc);
    return ns;
  }, [resource.waiting_queue]);

  return (
    <div className="rounded-xl bg-white p-5 shadow-sm">
      <div className="mb-4 flex items-center justify-between border-b border-zinc-100 pb-4">
        <div className="text-lg font-semibold text-zinc-900">{resource.name || resource.id}</div>
        <div className="flex items-center gap-2 text-xs">
          <span className="rounded-full bg-emerald-100 px-3 py-1 font-semibold text-emerald-800">
            Service: {serviceNodes.length}
          </span>
          <span className="rounded-full bg-amber-100 px-3 py-1 font-semibold text-amber-800">
            Waiting: {waitingNodes.length}
          </span>
        </div>
      </div>

      <div className="flex flex-col gap-4 md:flex-row">
        <div className="rounded-lg border border-zinc-200 bg-emerald-50/60 p-3 md:basis-[30%]">
          <div className="mb-2 text-xs font-semibold tracking-widest text-zinc-600">
            SERVICE ({serviceNodes.length})
          </div>
          {serviceNodes.length === 0 ? (
            <div className="py-6 text-center text-sm italic text-zinc-400">Empty</div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {serviceNodes.map((n) => (
                <NodeNamePill key={n.id} value={displayNodeName(n)} />
              ))}
            </div>
          )}
        </div>

        <div className="rounded-lg border border-zinc-200 bg-amber-50/60 p-3 md:basis-[70%]">
          <div className="mb-2 text-xs font-semibold tracking-widest text-zinc-600">
            WAITING ({waitingNodes.length})
          </div>
          {waitingNodes.length === 0 ? (
            <div className="py-6 text-center text-sm italic text-zinc-400">Empty</div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {waitingNodes.map((n) => (
                <NodeNamePill key={n.id} value={displayNodeName(n)} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export function WaitingRoomsPage() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<string | null>(null);
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);

  const refresh = useCallback(async (opts?: { log?: boolean }) => {
    const logCalls = opts?.log ?? true;
    try {
      const r = await listResources();
      setResources(r);
      setLastUpdated(nowTime());
    } catch (e) {
      if (logCalls) setToast({ kind: "error", message: (e as Error).message });
      throw e;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh({ log: true }).catch(() => {});
  }, [refresh]);

  useEffect(() => {
    const t = setInterval(() => {
      refresh({ log: false }).catch(() => {});
    }, 2000);
    return () => clearInterval(t);
  }, [refresh]);

  const sortedResources = useMemo(() => {
    const rs = resources.slice();
    rs.sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id));
    return rs;
  }, [resources]);

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Waiting Rooms</h1>
        <p className="mt-2 text-white/80">
          Rooms and services currently in use and waiting.
        </p>
      </header>

      {toast ? <Toast kind={toast.kind} message={toast.message} onClose={() => setToast(null)} /> : null}

      <div className="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-xl bg-white/10 px-4 py-3 text-sm text-white">
        <div className="font-semibold">{loading ? "Loading…" : "Live"}</div>
        <div className="text-white/80">{lastUpdated ? `Last updated: ${lastUpdated}` : ""}</div>
      </div>

      <section className="grid grid-cols-1 gap-6">
        {sortedResources.map((r) => (
          <RoomFrame key={r.id} resource={r} />
        ))}
        {!loading && sortedResources.length === 0 ? (
          <div className="rounded-xl bg-white p-6 text-center text-zinc-600">No rooms found.</div>
        ) : null}
      </section>
    </div>
  );
}

