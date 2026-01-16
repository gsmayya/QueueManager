"use client";

import React, { useMemo } from "react";
import type { ResourceSessionMetrics, ResourcesSessionMetricsResponse } from "../lib/types";

const EMPTY: ResourceSessionMetrics[] = [];

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

export function ResourceMetricsFrame({
  metrics,
  loading,
  error,
  lastUpdatedAt,
}: {
  metrics: ResourcesSessionMetricsResponse | null;
  loading: boolean;
  error: string | null;
  lastUpdatedAt: string | null;
}) {
  const rows = metrics?.resources ?? EMPTY;

  const sorted = useMemo(() => {
    return [...rows].sort((a, b) => (b.total_added ?? 0) - (a.total_added ?? 0));
  }, [rows]);

  return (
    <section className="mt-6 rounded-xl bg-white p-5 shadow-sm">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="text-lg font-semibold text-zinc-900">Resource Metrics</div>
        <div className="text-xs text-zinc-500">
          Refreshes every 10s
          {lastUpdatedAt ? <span className="ml-2">Last updated: {lastUpdatedAt}</span> : null}
        </div>
      </div>

      {loading ? (
        <div className="text-sm text-zinc-600">Loading metrics…</div>
      ) : error ? (
        <div className="text-sm text-red-600">{error}</div>
      ) : sorted.length === 0 ? (
        <div className="text-sm text-zinc-600">No resources.</div>
      ) : (
        <div className="overflow-auto rounded-lg border border-zinc-200">
          <table className="min-w-[900px] w-full text-left text-sm">
            <thead className="bg-zinc-50 text-xs font-semibold tracking-widest text-zinc-500">
              <tr>
                <th className="px-4 py-3">RESOURCE</th>
                <th className="px-4 py-3">CAPACITY</th>
                <th className="px-4 py-3">WAITING</th>
                <th className="px-4 py-3">ALLOCATED</th>
                <th className="px-4 py-3">ADDED</th>
                <th className="px-4 py-3">ALLOCATIONS</th>
                <th className="px-4 py-3">AVG_WAIT</th>
                <th className="px-4 py-3">AVG_SERVICE</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {sorted.map((r) => (
                <tr key={r.resource_id} className="hover:bg-zinc-50">
                  <td className="px-4 py-3 font-semibold text-zinc-900">{r.resource_name}</td>
                  <td className="px-4 py-3 font-semibold text-zinc-900">{r.resource_capacity}</td>
                  <td className="px-4 py-3 text-zinc-700">{r.current_waiting ?? 0}</td>
                  <td className="px-4 py-3 text-zinc-700">{r.current_allocated ?? 0}</td>
                  <td className="px-4 py-3 text-zinc-700">{r.total_added ?? 0}</td>
                  <td className="px-4 py-3 text-zinc-700">{r.total_allocated ?? 0}</td>
                  <td className="px-4 py-3 text-zinc-700">{formatDuration(r.avg_waiting_time_ms ?? 0)}</td>
                  <td className="px-4 py-3 text-zinc-700">{formatDuration(r.avg_service_time_ms ?? 0)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}


