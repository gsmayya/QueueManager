"use client";

import React, { useEffect, useMemo, useState } from "react";
import type { AdminEntity, ScheduleMetrics, SchedulesMetricsResponse } from "../lib/types";
import { getEntityById } from "../lib/masterApi";

function formatDuration(ms?: number): string {
  const v = ms ?? 0;
  if (!Number.isFinite(v) || v < 0) return "0s";
  const s = Math.floor(v / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = s % 60;
  if (h > 0) return `${h}h ${m}m ${ss}s`;
  if (m > 0) return `${m}m ${ss}s`;
  return `${ss}s`;
}

function pct(n: number, d: number): string {
  if (!d) return "0%";
  return `${Math.round((n / d) * 100)}%`;
}

export function ScheduleMetricsFrame({
  metrics,
  loading,
  error,
  lastUpdatedAt,
}: {
  metrics: SchedulesMetricsResponse | null;
  loading: boolean;
  error: string | null;
  lastUpdatedAt: string | null;
}) {
  const schedules = useMemo(() => metrics?.schedules ?? [], [metrics]);
  const totals = metrics?.totals;

  const [entityCache, setEntityCache] = useState<Record<string, AdminEntity>>({});

  // Cache entity names for display (best-effort).
  useEffect(() => {
    const ids = new Set<string>();
    for (const s of schedules) ids.add(s.entity_id);
    const missing = [...ids].filter((id) => !entityCache[id]);
    if (missing.length === 0) return;
    (async () => {
      const entries = await Promise.all(
        missing.map(async (id) => {
          try {
            const e = await getEntityById(id);
            return [id, e] as const;
          } catch {
            return null;
          }
        }),
      );
      const next: Record<string, AdminEntity> = {};
      for (const it of entries) {
        if (!it) continue;
        next[it[0]] = it[1];
      }
      if (Object.keys(next).length > 0) setEntityCache((prev) => ({ ...prev, ...next }));
    })();
  }, [schedules, entityCache]);

  const sorted = useMemo(() => {
    return [...schedules].sort((a, b) => (b.fired_count ?? 0) - (a.fired_count ?? 0));
  }, [schedules]);

  return (
    <section className="mt-6 rounded-xl bg-white p-5 shadow-sm">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="text-lg font-semibold text-zinc-900">Schedule Metrics</div>
        <div className="text-xs text-zinc-500">
          Refreshes every 10s
          {lastUpdatedAt ? <span className="ml-2">Last updated: {lastUpdatedAt}</span> : null}
        </div>
      </div>

      {loading ? (
        <div className="text-sm text-zinc-600">Loading metrics…</div>
      ) : error ? (
        <div className="text-sm text-red-600">{error}</div>
      ) : !totals ? (
        <div className="text-sm text-zinc-600">No schedule metrics.</div>
      ) : (
        <>
          <div className="mb-3 flex flex-wrap gap-2 text-sm">
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Schedules: <span className="font-semibold">{totals.total_schedules}</span>{" "}
              <span className="text-zinc-500">({totals.enabled_schedules} enabled, {totals.ended_schedules} ended)</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Fired: <span className="font-semibold">{totals.fired_count}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Completed: <span className="font-semibold">{totals.completed_count}</span>{" "}
              <span className="text-zinc-500">
                ({totals.completed_within_time_limit_count} within limit,{" "}
                {pct(totals.completed_within_time_limit_count, totals.completed_count)})
              </span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Expired: <span className="font-semibold">{totals.expired_count}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Avg assigned→allocate:{" "}
              <span className="font-semibold">{formatDuration(totals.avg_assigned_to_allocate_ms)}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Avg assigned→complete:{" "}
              <span className="font-semibold">{formatDuration(totals.avg_assigned_to_complete_ms)}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Avg assigned→expired:{" "}
              <span className="font-semibold">{formatDuration(totals.avg_assigned_to_expired_ms)}</span>
            </div>
          </div>

          {sorted.length === 0 ? (
            <div className="text-sm text-zinc-600">No schedules.</div>
          ) : (
            <div className="rounded-lg border border-zinc-200 overflow-x-auto">
              <table className="w-full min-w-[980px] table-fixed text-left text-xs">
                <colgroup>
                  <col style={{ width: "20%" }} />
                  <col style={{ width: "12%" }} />
                  <col style={{ width: "10%" }} />
                  <col style={{ width: "10%" }} />
                  <col style={{ width: "10%" }} />
                  <col style={{ width: "10%" }} />
                  <col style={{ width: "10%" }} />
                  <col style={{ width: "9%" }} />
                  <col style={{ width: "9%" }} />
                </colgroup>
                <thead className="bg-zinc-50 text-[11px] font-semibold tracking-widest text-zinc-500">
                  <tr>
                    <th className="px-2 py-2">ENTITY</th>
                    <th className="px-2 py-2">RESOURCE</th>
                    <th className="px-2 py-2">FIRED</th>
                    <th className="px-2 py-2">COMPLETED</th>
                    <th className="px-2 py-2">WITHIN_LIMIT</th>
                    <th className="px-2 py-2">EXPIRED</th>
                    <th className="px-2 py-2">A→ALLOC</th>
                    <th className="px-2 py-2">A→COMP</th>
                    <th className="px-2 py-2">A→EXP</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-200">
                  {sorted.map((s: ScheduleMetrics) => {
                    const ent = entityCache[s.entity_id];
                    const withinPct = pct(s.completed_within_time_limit_count, s.completed_count);
                    return (
                      <tr key={s.schedule_id}>
                        <td className="px-2 py-2">
                          <div className="font-semibold text-zinc-900">{ent?.name || s.entity_id}</div>
                          <div className="text-[11px] text-zinc-500">id: {s.schedule_id}</div>
                        </td>
                        <td className="px-2 py-2 font-mono text-zinc-900">{s.resource_id}</td>
                        <td className="px-2 py-2 text-zinc-700">{s.fired_count}</td>
                        <td className="px-2 py-2 text-zinc-700">{s.completed_count}</td>
                        <td className="px-2 py-2 text-zinc-700">
                          {s.completed_within_time_limit_count}{" "}
                          <span className="text-zinc-500">({withinPct})</span>
                        </td>
                        <td className="px-2 py-2 text-zinc-700">{s.expired_count}</td>
                        <td className="px-2 py-2 text-zinc-700">
                          {formatDuration(s.avg_assigned_to_allocate_ms)}
                        </td>
                        <td className="px-2 py-2 text-zinc-700">
                          {formatDuration(s.avg_assigned_to_complete_ms)}
                        </td>
                        <td className="px-2 py-2 text-zinc-700">
                          {formatDuration(s.avg_assigned_to_expired_ms)}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </section>
  );
}

