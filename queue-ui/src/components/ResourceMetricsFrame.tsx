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
        <div className="rounded-lg border border-zinc-200">
          <table className="w-full table-fixed text-left text-xs">
            <colgroup>
              <col style={{ width: "15%" }} />
              <col style={{ width: "10%" }} />
              {/* Current (2) */}
              <col style={{ width: "9.75%" }} />
              <col style={{ width: "9.75%" }} />
              {/* Total (4) */}
              <col style={{ width: "9.75%" }} />
              <col style={{ width: "9.75%" }} />
              <col style={{ width: "9.75%" }} />
              <col style={{ width: "9.75%" }} />
              {/* Average (2) */}
              <col style={{ width: "9.75%" }} />
              <col style={{ width: "9.75%" }} />
            </colgroup>

            <thead className="text-[11px] font-semibold tracking-widest text-zinc-500">
              <tr>
                <th className="bg-zinc-50 px-2 py-2" rowSpan={2}>
                  RESOURCE
                </th>
                <th className="bg-zinc-50 px-2 py-2" rowSpan={2}>
                  CAPACITY
                </th>
                <th className="bg-emerald-50 px-2 py-2 text-center" colSpan={2}>
                  CURRENT
                </th>
                <th className="bg-amber-50 px-2 py-2 text-center" colSpan={2}>
                  AVERAGE
                </th>
                <th className="bg-sky-50 px-2 py-2 text-center" colSpan={4}>
                  TOTAL
                </th>
                
              </tr>
              <tr className="text-[10px] [&_th]:px-2 [&_th]:pb-2 [&_th]:pt-0">
                <th className="bg-emerald-50">Waiting</th>
                <th className="bg-emerald-50">Allocated</th>
                <th className="bg-amber-50">Wait time</th>
                <th className="bg-amber-50">Service time</th>
                <th className="bg-sky-50">Waiting</th>
                <th className="bg-sky-50">Allocated</th>
                <th className="bg-sky-50">Wait time</th>
                <th className="bg-sky-50">Service time</th>                
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200">
              {sorted.map((r) => (
                <tr key={r.resource_id}>
                  <td className="bg-white px-2 py-2 font-semibold text-zinc-900">{r.resource_name}</td>
                  <td className="bg-white px-2 py-2 font-semibold text-zinc-900">{r.resource_capacity}</td>
                  {/* Current */}
                  <td className="bg-emerald-50 px-2 py-2 text-zinc-700">{r.current_waiting ?? 0}</td>
                  <td className="bg-emerald-50 px-2 py-2 text-zinc-700">{r.current_allocated ?? 0}</td>


                  {/* Average */}
                  <td className="bg-amber-50 px-2 py-2 text-zinc-700">{formatDuration(r.avg_waiting_time_ms ?? 0)}</td>
                  <td className="bg-amber-50 px-2 py-2 text-zinc-700">{formatDuration(r.avg_service_time_ms ?? 0)}</td>

                  {/* Total */}
                  <td className="bg-sky-50 px-2 py-2 text-zinc-700">{r.total_added ?? 0}</td>
                  <td className="bg-sky-50 px-2 py-2 text-zinc-700">{r.total_allocated ?? 0}</td>
                  <td className="bg-sky-50 px-2 py-2 text-zinc-700">{formatDuration(r.waiting_total_ms ?? 0)}</td>
                  <td className="bg-sky-50 px-2 py-2 text-zinc-700">{formatDuration(r.service_total_ms ?? 0)}</td>

                  
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}


