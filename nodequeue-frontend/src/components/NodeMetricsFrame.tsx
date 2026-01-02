"use client";

import React, { useMemo } from "react";
import type { NodeMetrics, NodesMetricsResponse } from "../lib/types";

const EMPTY_NODES: NodeMetrics[] = [];

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

function sumWaitingMs(n: NodeMetrics): number {
  return (n.waiting_segments || []).reduce((acc, seg) => acc + (seg.duration_ms || 0), 0);
}

function waitingByResource(n: NodeMetrics): Array<[string, number]> {
  const map = new Map<string, number>();
  for (const seg of n.waiting_segments || []) {
    map.set(seg.resource_id, (map.get(seg.resource_id) ?? 0) + (seg.duration_ms || 0));
  }
  return [...map.entries()].sort((a, b) => b[1] - a[1]);
}

export function NodeMetricsFrame({
  metrics,
  loading,
  error,
  lastUpdatedAt,
}: {
  metrics: NodesMetricsResponse | null;
  loading: boolean;
  error: string | null;
  lastUpdatedAt: string | null;
}) {
  const active = metrics?.active_nodes ?? EMPTY_NODES;
  const completed = metrics?.completed_nodes ?? EMPTY_NODES;

  const topActive = useMemo(() => {
    // Show a compact list: active nodes sorted by longest time-in-system.
    return [...active].sort((a, b) => (b.total_time_in_system_ms ?? 0) - (a.total_time_in_system_ms ?? 0));
  }, [active]);

  return (
    <section className="mt-6 rounded-xl bg-white p-5 shadow-sm">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="text-lg font-semibold text-zinc-900">Node Metrics</div>
        <div className="text-xs text-zinc-500">
          Refreshes every 10s
          {lastUpdatedAt ? <span className="ml-2">Last updated: {lastUpdatedAt}</span> : null}
        </div>
      </div>

      {loading ? (
        <div className="text-sm text-zinc-600">Loading metrics…</div>
      ) : error ? (
        <div className="text-sm text-red-600">{error}</div>
      ) : (
        <>
          <div className="mb-3 flex flex-wrap gap-2 text-sm">
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Active: <span className="font-semibold">{active.length}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Completed: <span className="font-semibold">{completed.length}</span>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div>
              <div className="mb-2 text-xs font-semibold tracking-widest text-zinc-500">ACTIVE</div>
              {topActive.length === 0 ? (
                <div className="text-sm text-zinc-600">No active nodes.</div>
              ) : (
                <div className="space-y-2">
                  {topActive.map((n) => (
                    <div key={n.id} className="rounded-lg border border-zinc-200 p-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="font-semibold text-zinc-900">{n.entity_name}</div>
                        <div className="text-xs text-zinc-500 font-mono">{n.id}</div>
                      </div>
                      <div className="mt-1 text-sm text-zinc-700">
                        Total: <span className="font-semibold">{formatDuration(n.total_time_in_system_ms)}</span>
                        <span className="ml-3">
                          Waiting: <span className="font-semibold">{formatDuration(sumWaitingMs(n))}</span>
                        </span>
                      </div>
                      {(n.waiting_segments?.length ?? 0) > 0 ? (
                        <div className="mt-2 flex flex-wrap gap-2">
                          {n.waiting_segments.map((seg, idx) => (
                            <div
                              key={`${n.id}-${idx}`}
                              className="rounded-full bg-amber-50 px-3 py-1 text-xs text-amber-800"
                              title={`${seg.resource_id}: ${seg.start_ts} → ${seg.end_ts}`}
                            >
                              {seg.resource_id}: {formatDuration(seg.duration_ms)}
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div>
              <div className="mb-2 text-xs font-semibold tracking-widest text-zinc-500">COMPLETED</div>
              {completed.length === 0 ? (
                <div className="text-sm text-zinc-600">No completed nodes.</div>
              ) : (
                <div className="max-h-[360px] space-y-2 overflow-auto rounded-lg border border-zinc-200 p-3">
                  {completed.map((n) => (
                    <div key={n.id} className="rounded-lg border border-zinc-100 p-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="font-semibold text-zinc-900">{n.entity_name}</div>
                        <div className="text-xs text-zinc-500 font-mono">{n.id}</div>
                      </div>
                      <div className="mt-1 text-sm text-zinc-700">
                        Total: <span className="font-semibold">{formatDuration(n.total_time_in_system_ms)}</span>
                        <span className="ml-3">
                          Waiting: <span className="font-semibold">{formatDuration(sumWaitingMs(n))}</span>
                        </span>
                      </div>

                      {waitingByResource(n).length > 0 ? (
                        <div className="mt-2 flex flex-wrap gap-2">
                          {waitingByResource(n).map(([rid, ms]) => (
                            <div
                              key={`${n.id}-${rid}`}
                              className="rounded-full bg-amber-50 px-3 py-1 text-xs text-amber-800"
                            >
                              {rid}: {formatDuration(ms)}
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </section>
  );
}


