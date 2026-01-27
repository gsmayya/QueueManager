"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import type { AdminEntity, Resource, Schedule } from "../../lib/types";
import { createSchedule, listResources, listSchedules, updateSchedule } from "../../lib/api";
import { getEntityById, listEntitiesByPhone } from "../../lib/masterApi";
import { Toast } from "../Toast";

function nowTime(): string {
  return new Date().toLocaleTimeString();
}

function toDateTimeLocalValue(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (!Number.isFinite(d.getTime())) return "";
  // datetime-local expects YYYY-MM-DDTHH:mm
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function fromDateTimeLocalValue(v: string): string | undefined {
  const s = (v || "").trim();
  if (!s) return undefined;
  const d = new Date(s);
  if (!Number.isFinite(d.getTime())) return undefined;
  return d.toISOString();
}

type EditDraft = {
  resource_id: string;
  interval_seconds: string;
  time_limit_seconds: string;
  waiting_expiry_seconds: string;
  ends_at: string; // datetime-local input value
};

export function SchedulesPage() {
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);

  const [resources, setResources] = useState<Resource[]>([]);
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<string | null>(null);

  // Create form state (select entity via phone search like NodePage)
  const [phone, setPhone] = useState("");
  const [searchLoading, setSearchLoading] = useState(false);
  const [matches, setMatches] = useState<AdminEntity[]>([]);
  const [selectedEntityId, setSelectedEntityId] = useState<string>("");

  const [resourceId, setResourceId] = useState<string>("");
  const [intervalSeconds, setIntervalSeconds] = useState<string>("60");
  const [timeLimitSeconds, setTimeLimitSeconds] = useState<string>("300");
  const [waitingExpirySeconds, setWaitingExpirySeconds] = useState<string>("900");
  const [endsAt, setEndsAt] = useState<string>(""); // datetime-local
  const [enabled, setEnabled] = useState<boolean>(true);

  // Entity cache for display
  const [entityCache, setEntityCache] = useState<Record<string, AdminEntity>>({});

  // Editing
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<EditDraft | null>(null);

  const resourceOptions = useMemo(
    () => resources.map((r) => ({ id: r.id, name: r.name || r.id, capacity: r.capacity })),
    [resources],
  );

  const refresh = useCallback(async (opts?: { log?: boolean }) => {
    const logCalls = opts?.log ?? true;
    try {
      const [r, s] = await Promise.all([listResources(), listSchedules()]);
      setResources(r);
      setSchedules(s);
      setLastUpdated(nowTime());
    } catch (e) {
      const err = e as Error;
      if (logCalls) setToast({ kind: "error", message: err.message });
      throw e;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh({ log: true }).catch(() => {});
  }, [refresh]);

  // Cache entity details for displayed schedules.
  useEffect(() => {
    const ids = new Set<string>();
    for (const s of schedules) {
      if (s.entity_id) ids.add(s.entity_id);
    }
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
      if (Object.keys(next).length > 0) {
        setEntityCache((prev) => ({ ...prev, ...next }));
      }
    })();
  }, [schedules, entityCache]);

  const onSearch = useCallback(async () => {
    const p = phone.trim();
    if (!p) return;
    setSearchLoading(true);
    try {
      const res = await listEntitiesByPhone(p);
      setMatches(res);
      if (res.length === 1) setSelectedEntityId(res[0].id);
    } catch (e) {
      const err = e as Error;
      setToast({ kind: "error", message: err.message });
    } finally {
      setSearchLoading(false);
    }
  }, [phone]);

  const onCreate = useCallback(async () => {
    const entId = selectedEntityId.trim();
    const rid = resourceId.trim();
    const interval = Number(intervalSeconds);
    const limit = Number(timeLimitSeconds);
    const waitExp = Number(waitingExpirySeconds);
    const endsAtIso = fromDateTimeLocalValue(endsAt);
    if (!entId) {
      setToast({ kind: "error", message: "Select an entity first" });
      return;
    }
    if (!rid) {
      setToast({ kind: "error", message: "Select a resource first" });
      return;
    }
    if (!Number.isFinite(interval) || interval <= 0) {
      setToast({ kind: "error", message: "interval_seconds must be > 0" });
      return;
    }
    if (!Number.isFinite(limit) || limit <= 0) {
      setToast({ kind: "error", message: "time_limit_seconds must be > 0" });
      return;
    }
    if (!Number.isFinite(waitExp) || waitExp <= 0) {
      setToast({ kind: "error", message: "waiting_expiry_seconds must be > 0" });
      return;
    }
    if (endsAtIso) {
      const d = new Date(endsAtIso);
      if (d.getTime() <= Date.now()) {
        setToast({ kind: "error", message: "ends_at must be in the future" });
        return;
      }
    }
    try {
      await createSchedule({
        entity_id: entId,
        resource_id: rid,
        interval_seconds: interval,
        time_limit_seconds: limit,
        waiting_expiry_seconds: waitExp,
        ...(endsAtIso ? { ends_at: endsAtIso } : {}),
        enabled,
      });
      setToast({ kind: "success", message: "Schedule created" });
      setPhone("");
      setMatches([]);
      setSelectedEntityId("");
      setResourceId("");
      setEndsAt("");
      await refresh({ log: false });
    } catch (e) {
      const err = e as Error;
      setToast({ kind: "error", message: err.message });
    }
  }, [selectedEntityId, resourceId, intervalSeconds, timeLimitSeconds, waitingExpirySeconds, endsAt, enabled, refresh]);

  const onToggleEnabled = useCallback(
    async (s: Schedule) => {
      try {
        await updateSchedule(s.id, { enabled: !s.enabled });
        setToast({ kind: "success", message: `Schedule ${!s.enabled ? "enabled" : "disabled"}` });
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        setToast({ kind: "error", message: err.message });
      }
    },
    [refresh],
  );

  const startEdit = useCallback((s: Schedule) => {
    setEditingId(s.id);
    setDraft({
      resource_id: s.resource_id,
      interval_seconds: String(s.interval_seconds),
      time_limit_seconds: String(s.time_limit_seconds),
      waiting_expiry_seconds: String(s.waiting_expiry_seconds),
      ends_at: toDateTimeLocalValue(s.ends_at),
    });
  }, []);

  const cancelEdit = useCallback(() => {
    setEditingId(null);
    setDraft(null);
  }, []);

  const saveEdit = useCallback(
    async (s: Schedule) => {
      if (!draft) return;
      const rid = draft.resource_id.trim();
      const interval = Number(draft.interval_seconds);
      const limit = Number(draft.time_limit_seconds);
      const waitExp = Number(draft.waiting_expiry_seconds);
      const endsAtIso = fromDateTimeLocalValue(draft.ends_at);
      if (!rid) {
        setToast({ kind: "error", message: "resource_id is required" });
        return;
      }
      if (!Number.isFinite(interval) || interval <= 0) {
        setToast({ kind: "error", message: "interval_seconds must be > 0" });
        return;
      }
      if (!Number.isFinite(limit) || limit <= 0) {
        setToast({ kind: "error", message: "time_limit_seconds must be > 0" });
        return;
      }
      if (!Number.isFinite(waitExp) || waitExp <= 0) {
        setToast({ kind: "error", message: "waiting_expiry_seconds must be > 0" });
        return;
      }
      if (endsAtIso) {
        const d = new Date(endsAtIso);
        if (d.getTime() <= Date.now()) {
          setToast({ kind: "error", message: "ends_at must be in the future" });
          return;
        }
      }
      try {
        await updateSchedule(s.id, {
          resource_id: rid,
          interval_seconds: interval,
          time_limit_seconds: limit,
          waiting_expiry_seconds: waitExp,
          ...(endsAtIso ? { ends_at: endsAtIso } : {}),
        });
        setToast({ kind: "success", message: "Schedule updated" });
        cancelEdit();
        await refresh({ log: false });
      } catch (e) {
        const err = e as Error;
        setToast({ kind: "error", message: err.message });
      }
    },
    [draft, refresh, cancelEdit],
  );

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Schedules</h1>
        <p className="mt-2 text-white/80">Create recurring schedules and manage interval/time limits.</p>
        <nav className="mt-4 flex justify-center gap-3 text-sm">
          <Link href="/queue" className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15">
            Manage Queue
          </Link>
          <Link href="/node" className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15">
            Nodes
          </Link>
          <Link href="/metrics" className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15">
            Metrics
          </Link>
          <Link href="/admin" className="rounded-full bg-white/10 px-4 py-2 font-semibold text-white hover:bg-white/15">
            Admin
          </Link>
        </nav>
      </header>

      {toast ? <Toast kind={toast.kind} message={toast.message} onClose={() => setToast(null)} /> : null}

      <section className="mb-6 rounded-xl bg-white p-5 shadow-sm">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <div className="text-lg font-semibold text-zinc-900">Create schedule</div>
            <div className="text-sm text-zinc-600">Pick an entity, a resource, and set interval + time limit.</div>
          </div>
          <button
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-semibold text-white hover:bg-zinc-800"
            onClick={() => refresh({ log: true }).catch(() => {})}
            disabled={loading}
          >
            Refresh{lastUpdated ? ` (${lastUpdated})` : ""}
          </button>
        </div>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div className="rounded-lg border border-zinc-200 p-4">
            <div className="mb-2 text-sm font-semibold text-zinc-900">1) Find entity</div>
            <div className="flex gap-2">
              <input
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="Search by phone (e.g., +1... or 555...)"
                className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-indigo-500 focus:outline-none"
              />
              <button
                className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:bg-zinc-300"
                onClick={() => onSearch().catch(() => {})}
                disabled={searchLoading}
              >
                Search
              </button>
            </div>
            <div className="mt-3">
              <select
                value={selectedEntityId}
                onChange={(e) => setSelectedEntityId(e.target.value)}
                className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
              >
                <option value="">Select entity…</option>
                {matches.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name} ({m.phone})
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="rounded-lg border border-zinc-200 p-4">
            <div className="mb-2 text-sm font-semibold text-zinc-900">2) Schedule settings</div>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <div className="md:col-span-2">
                <label className="mb-1 block text-xs font-semibold text-zinc-700">Resource</label>
                <select
                  value={resourceId}
                  onChange={(e) => setResourceId(e.target.value)}
                  className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
                >
                  <option value="">Select resource…</option>
                  {resourceOptions.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.name} ({r.id}) — capacity {r.capacity}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="mb-1 block text-xs font-semibold text-zinc-700">Interval (seconds)</label>
                <input
                  value={intervalSeconds}
                  onChange={(e) => setIntervalSeconds(e.target.value)}
                  inputMode="numeric"
                  className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
                />
              </div>

              <div>
                <label className="mb-1 block text-xs font-semibold text-zinc-700">Time limit (seconds)</label>
                <input
                  value={timeLimitSeconds}
                  onChange={(e) => setTimeLimitSeconds(e.target.value)}
                  inputMode="numeric"
                  className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
                />
              </div>

              <div>
                <label className="mb-1 block text-xs font-semibold text-zinc-700">Waiting expiry (seconds)</label>
                <input
                  value={waitingExpirySeconds}
                  onChange={(e) => setWaitingExpirySeconds(e.target.value)}
                  inputMode="numeric"
                  className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
                />
              </div>

              <div className="md:col-span-2">
                <label className="mb-1 block text-xs font-semibold text-zinc-700">Schedule ends at (optional)</label>
                <input
                  type="datetime-local"
                  value={endsAt}
                  onChange={(e) => setEndsAt(e.target.value)}
                  className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none"
                />
              </div>

              <div className="md:col-span-2 flex items-center justify-between">
                <label className="flex items-center gap-2 text-sm text-zinc-800">
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(e) => setEnabled(e.target.checked)}
                    className="h-4 w-4"
                  />
                  Enabled
                </label>

                <button
                  className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-semibold text-white hover:bg-zinc-800"
                  onClick={() => onCreate().catch(() => {})}
                  disabled={loading}
                >
                  Create schedule
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="rounded-xl bg-white p-5 shadow-sm">
        <div className="mb-3 flex items-center justify-between">
          <div>
            <div className="text-lg font-semibold text-zinc-900">Existing schedules</div>
            <div className="text-sm text-zinc-600">Toggle enabled or edit settings.</div>
          </div>
          <div className="text-sm text-zinc-500">{schedules.length} total</div>
        </div>

        {schedules.length === 0 ? (
          <div className="text-sm text-zinc-600">No schedules yet.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full table-auto text-left text-sm">
              <thead>
                <tr className="border-b border-zinc-200 text-xs font-semibold uppercase tracking-wider text-zinc-600">
                  <th className="py-2 pr-3">Entity</th>
                  <th className="py-2 pr-3">Resource</th>
                  <th className="py-2 pr-3">Interval</th>
                  <th className="py-2 pr-3">Time limit</th>
                  <th className="py-2 pr-3">Wait expiry</th>
                  <th className="py-2 pr-3">Ends at</th>
                  <th className="py-2 pr-3">Next run</th>
                  <th className="py-2 pr-3">Status</th>
                  <th className="py-2 pr-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {schedules.map((s) => {
                  const ent = entityCache[s.entity_id];
                  const isEditing = editingId === s.id;
                  return (
                    <tr key={s.id} className="border-b border-zinc-100 align-top">
                      <td className="py-3 pr-3">
                        <div className="font-semibold text-zinc-900">{ent?.name || s.entity_id}</div>
                        <div className="text-xs text-zinc-500">{ent?.phone || ""}</div>
                      </td>

                      <td className="py-3 pr-3">
                        {isEditing && draft ? (
                          <select
                            value={draft.resource_id}
                            onChange={(e) => setDraft({ ...draft, resource_id: e.target.value })}
                            className="w-56 rounded-md border border-zinc-200 bg-white px-2 py-1 text-sm text-zinc-900"
                          >
                            {resourceOptions.map((r) => (
                              <option key={r.id} value={r.id}>
                                {r.name} ({r.id})
                              </option>
                            ))}
                          </select>
                        ) : (
                          <span className="font-mono text-zinc-900">{s.resource_id}</span>
                        )}
                      </td>

                      <td className="py-3 pr-3">
                        {isEditing && draft ? (
                          <input
                            value={draft.interval_seconds}
                            onChange={(e) => setDraft({ ...draft, interval_seconds: e.target.value })}
                            className="w-28 rounded-md border border-zinc-200 bg-white px-2 py-1 text-sm text-zinc-900"
                          />
                        ) : (
                          <span>{s.interval_seconds}s</span>
                        )}
                      </td>

                      <td className="py-3 pr-3">
                        {isEditing && draft ? (
                          <input
                            value={draft.time_limit_seconds}
                            onChange={(e) => setDraft({ ...draft, time_limit_seconds: e.target.value })}
                            className="w-28 rounded-md border border-zinc-200 bg-white px-2 py-1 text-sm text-zinc-900"
                          />
                        ) : (
                          <span>{s.time_limit_seconds}s</span>
                        )}
                      </td>

                      <td className="py-3 pr-3">
                        {isEditing && draft ? (
                          <input
                            value={draft.waiting_expiry_seconds}
                            onChange={(e) => setDraft({ ...draft, waiting_expiry_seconds: e.target.value })}
                            className="w-28 rounded-md border border-zinc-200 bg-white px-2 py-1 text-sm text-zinc-900"
                          />
                        ) : (
                          <span>{s.waiting_expiry_seconds}s</span>
                        )}
                      </td>

                      <td className="py-3 pr-3">
                        {isEditing && draft ? (
                          <input
                            type="datetime-local"
                            value={draft.ends_at}
                            onChange={(e) => setDraft({ ...draft, ends_at: e.target.value })}
                            className="w-56 rounded-md border border-zinc-200 bg-white px-2 py-1 text-sm text-zinc-900"
                          />
                        ) : s.ends_at ? (
                          <span className="font-mono text-xs text-zinc-700">{new Date(s.ends_at).toLocaleString()}</span>
                        ) : (
                          <span className="text-xs text-zinc-500">—</span>
                        )}
                      </td>

                      <td className="py-3 pr-3">
                        <span className="font-mono text-xs text-zinc-700">{new Date(s.next_run_at).toLocaleString()}</span>
                      </td>

                      <td className="py-3 pr-3">
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-semibold ${
                            s.enabled ? "bg-emerald-50 text-emerald-700" : "bg-zinc-100 text-zinc-700"
                          }`}
                        >
                          {s.enabled ? "Enabled" : "Disabled"}
                        </span>
                      </td>

                      <td className="py-3 pr-3">
                        <div className="flex flex-wrap gap-2">
                          <button
                            className="rounded-md bg-white px-3 py-1 text-xs font-semibold text-zinc-900 ring-1 ring-zinc-200 hover:bg-zinc-50"
                            onClick={() => onToggleEnabled(s).catch(() => {})}
                          >
                            {s.enabled ? "Disable" : "Enable"}
                          </button>

                          {isEditing ? (
                            <>
                              <button
                                className="rounded-md bg-zinc-900 px-3 py-1 text-xs font-semibold text-white hover:bg-zinc-800"
                                onClick={() => saveEdit(s).catch(() => {})}
                                disabled={!draft}
                              >
                                Save
                              </button>
                              <button
                                className="rounded-md bg-white px-3 py-1 text-xs font-semibold text-zinc-900 ring-1 ring-zinc-200 hover:bg-zinc-50"
                                onClick={cancelEdit}
                              >
                                Cancel
                              </button>
                            </>
                          ) : (
                            <button
                              className="rounded-md bg-white px-3 py-1 text-xs font-semibold text-zinc-900 ring-1 ring-zinc-200 hover:bg-zinc-50"
                              onClick={() => startEdit(s)}
                            >
                              Edit
                            </button>
                          )}
                        </div>
                        <div className="mt-1 text-[11px] text-zinc-500">id: {s.id}</div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

