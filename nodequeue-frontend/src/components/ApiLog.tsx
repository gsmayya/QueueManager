"use client";

import React from "react";

export type ApiLogEntry = {
  time: string;
  method: string;
  url: string;
  status: number;
  body?: unknown;
  error?: string;
};

export function ApiLog({
  entries,
  onClear,
}: {
  entries: ApiLogEntry[];
  onClear: () => void;
}) {
  return (
    <section className="mt-6 rounded-xl bg-zinc-900 p-5 text-zinc-200 shadow-sm">
      <div className="mb-3 flex items-center justify-between border-b border-zinc-700 pb-3">
        <div className="font-semibold text-white">API Call Log</div>
        <button
          className="rounded-md bg-zinc-700 px-3 py-1.5 text-sm text-white hover:bg-zinc-600"
          onClick={onClear}
          type="button"
        >
          Clear Log
        </button>
      </div>

      <div className="max-h-[420px] overflow-y-auto">
        {entries.length === 0 ? (
          <div className="py-6 text-center text-sm text-zinc-400">
            No API calls yet.
          </div>
        ) : (
          [...entries].reverse().map((e, idx) => {
            const ok = e.status >= 200 && e.status < 300;
            return (
              <div
                key={`${e.time}-${idx}`}
                className="border-b border-zinc-800 py-2 last:border-b-0"
              >
                <div className="flex flex-wrap items-center gap-2 text-sm">
                  <span className="text-zinc-400">{e.time}</span>
                  <span
                    className={
                      e.method.toUpperCase() === "GET"
                        ? "font-semibold text-emerald-400"
                        : "font-semibold text-sky-400"
                    }
                  >
                    {e.method.toUpperCase()}
                  </span>
                  <span className="text-sky-200">{e.url}</span>
                  <span
                    className={
                      ok
                        ? "rounded bg-emerald-600 px-2 py-0.5 text-white"
                        : "rounded bg-rose-600 px-2 py-0.5 text-white"
                    }
                  >
                    {e.status}
                  </span>
                </div>
                {e.error ? (
                  <div className="mt-1 ml-5 text-sm text-rose-300">
                    ERROR: {e.error}
                  </div>
                ) : e.body && e.method.toUpperCase() === "POST" ? (
                  <div className="mt-1 ml-5 text-sm text-amber-300">
                    Body: {JSON.stringify(e.body)}
                  </div>
                ) : null}
              </div>
            );
          })
        )}
      </div>
    </section>
  );
}


