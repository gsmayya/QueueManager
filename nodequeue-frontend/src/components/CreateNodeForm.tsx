"use client";

import React, { useMemo, useState } from "react";
import type { Resource } from "../lib/types";

export function CreateNodeForm({
  resources,
  onCreate,
  disabled,
}: {
  resources: Resource[];
  onCreate: (args: { entityName: string; resourceId?: string }) => void;
  disabled?: boolean;
}) {
  const [entityName, setEntityName] = useState("");
  const [resourceId, setResourceId] = useState("");

  const resourceOptions = useMemo(
    () => resources.map((r) => ({ id: r.id, capacity: r.capacity })),
    [resources],
  );

  return (
    <section className="mb-6 rounded-xl bg-white p-5 shadow-sm">
      <form
        className="flex flex-col gap-3 md:flex-row md:items-center"
        onSubmit={(e) => {
          e.preventDefault();
          if (disabled) return;
          const name = entityName.trim();
          if (!name) return;
          onCreate({ entityName: name, resourceId: resourceId || undefined });
          setEntityName("");
          setResourceId("");
        }}
      >
        <input
          value={entityName}
          onChange={(e) => setEntityName(e.target.value)}
          placeholder="Node name (e.g., task-1)"
          className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-indigo-500 focus:outline-none"
          disabled={disabled}
        />

        <select
          value={resourceId}
          onChange={(e) => setResourceId(e.target.value)}
          className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-indigo-500 focus:outline-none md:w-80"
          disabled={disabled}
        >
          <option value="">Select resource (optional)</option>
          {resourceOptions.map((r) => (
            <option key={r.id} value={r.id}>
              {r.id} (Capacity: {r.capacity})
            </option>
          ))}
        </select>

        <button
          type="submit"
          disabled={disabled}
          className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:bg-zinc-300"
        >
          Create Node
        </button>
      </form>
    </section>
  );
}


