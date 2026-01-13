"use client";

import React from "react";

export function Toast({
  kind,
  message,
  onClose,
}: {
  kind: "error" | "success";
  message: string;
  onClose: () => void;
}) {
  const cls =
    kind === "error"
      ? "bg-rose-50 text-rose-800 border-rose-200"
      : "bg-emerald-50 text-emerald-800 border-emerald-200";

  return (
    <div className={`mb-4 rounded-lg border px-4 py-3 text-sm ${cls}`}>
      <div className="flex items-start justify-between gap-4">
        <div>{message}</div>
        <button
          type="button"
          onClick={onClose}
          className="rounded px-2 py-0.5 text-xs hover:bg-black/5"
        >
          x
        </button>
      </div>
    </div>
  );
}


