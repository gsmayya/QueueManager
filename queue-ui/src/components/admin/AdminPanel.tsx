"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import type { AdminEntity, AdminRoom, AdminUser } from "../../lib/types";
import {
  createAdminEntity,
  createAdminRoom,
  createAdminUser,
  deleteAdminEntity,
  deleteAdminRoom,
  deleteAdminUser,
  listAdminEntities,
  listAdminRooms,
  listAdminUsers,
  updateAdminEntity,
  updateAdminRoom,
  updateAdminUser,
} from "../../lib/masterApi";
import { Toast } from "../Toast";

type Tab = "entities" | "users" | "rooms";

function nowTime(): string {
  return new Date().toLocaleTimeString();
}

export function AdminPanel() {
  const [tab, setTab] = useState<Tab>("entities");
  const [toast, setToast] = useState<{ kind: "error" | "success"; message: string } | null>(null);

  const [entities, setEntities] = useState<AdminEntity[]>([]);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [rooms, setRooms] = useState<AdminRoom[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdatedAt, setLastUpdatedAt] = useState<string | null>(null);
  const [includeDeletedRooms, setIncludeDeletedRooms] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [e, u, r] = await Promise.all([
        listAdminEntities(),
        listAdminUsers(),
        listAdminRooms({ includeDeleted: includeDeletedRooms }),
      ]);
      setEntities(e);
      setUsers(u);
      setRooms(r);
      setLastUpdatedAt(nowTime());
    } catch (err) {
      const e = err as Error;
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [includeDeletedRooms]);

  useEffect(() => {
    refresh().catch(() => {});
  }, [refresh]);

  const roomsByStatus = useMemo(() => {
    const active: AdminRoom[] = [];
    const deleted: AdminRoom[] = [];
    for (const r of rooms) (r.deleted_at ? deleted : active).push(r);
    return { active, deleted };
  }, [rooms]);

  // --- Entities actions

  const onCreateEntity = useCallback(async () => {
    const name = window.prompt("Customer name?");
    if (!name) return;
    const phone = window.prompt("Phone number?");
    if (!phone) return;
    const email = window.prompt("Email (optional)?") || undefined;
    try {
      await createAdminEntity({ name, phone, email });
      setToast({ kind: "success", message: "Entity created" });
      await refresh();
    } catch (e) {
      setToast({ kind: "error", message: (e as Error).message });
    }
  }, [refresh]);

  const onEditEntity = useCallback(
    async (ent: AdminEntity) => {
      const name = window.prompt("New name?", ent.name);
      if (name === null) return;
      const phone = window.prompt("New phone?", ent.phone);
      if (phone === null) return;
      const email = window.prompt("New email (optional)?", ent.email || "");
      if (email === null) return;
      try {
        await updateAdminEntity(ent.id, { name, phone, email });
        setToast({ kind: "success", message: "Entity updated" });
        await refresh();
      } catch (e) {
        setToast({ kind: "error", message: (e as Error).message });
      }
    },
    [refresh],
  );

  const onDeleteEntity = useCallback(
    async (ent: AdminEntity) => {
      const ok = window.confirm(`Delete entity "${ent.name}"?`);
      if (!ok) return;
      try {
        await deleteAdminEntity(ent.id);
        setToast({ kind: "success", message: "Entity deleted" });
        await refresh();
      } catch (e) {
        setToast({ kind: "error", message: (e as Error).message });
      }
    },
    [refresh],
  );

  // --- Users actions

  const onCreateUser = useCallback(async () => {
    const userId = window.prompt("user_id?");
    if (!userId) return;
    const name = window.prompt("Name?");
    if (!name) return;
    const email = window.prompt("Email?");
    if (!email) return;
    const password = window.prompt("Password?");
    if (!password) return;
    try {
      await createAdminUser({ user_id: userId, name, email, password });
      setToast({ kind: "success", message: "User created" });
      await refresh();
    } catch (e) {
      setToast({ kind: "error", message: (e as Error).message });
    }
  }, [refresh]);

  const onEditUser = useCallback(
    async (u: AdminUser) => {
      const user_id = window.prompt("New user_id?", u.user_id);
      if (user_id === null) return;
      const name = window.prompt("New name?", u.name);
      if (name === null) return;
      const email = window.prompt("New email?", u.email);
      if (email === null) return;
      const password = window.prompt("New password (leave blank to keep current)?") ?? "";
      const patch: Partial<{ user_id: string; name: string; email: string; password: string }> = { user_id, name, email };
      if (password.trim()) patch.password = password;
      try {
        await updateAdminUser(u.id, patch);
        setToast({ kind: "success", message: "User updated" });
        await refresh();
      } catch (e) {
        setToast({ kind: "error", message: (e as Error).message });
      }
    },
    [refresh],
  );

  const onDeleteUser = useCallback(
    async (u: AdminUser) => {
      const ok = window.confirm(`Delete user "${u.user_id}"?`);
      if (!ok) return;
      try {
        await deleteAdminUser(u.id);
        setToast({ kind: "success", message: "User deleted" });
        await refresh();
      } catch (e) {
        setToast({ kind: "error", message: (e as Error).message });
      }
    },
    [refresh],
  );

  // --- Rooms actions

  const onCreateRoom = useCallback(async () => {
    const name = window.prompt("Room name?");
    if (!name) return;
    const capStr = window.prompt("Capacity?", "1");
    if (capStr === null) return;
    const capacity = Number(capStr);
    if (!Number.isFinite(capacity) || capacity < 0) {
      setToast({ kind: "error", message: "Invalid capacity" });
      return;
    }
    try {
      await createAdminRoom({ name, capacity });
      setToast({ kind: "success", message: "Room created" });
      await refresh();
    } catch (e) {
      setToast({ kind: "error", message: (e as Error).message });
    }
  }, [refresh]);

  const onEditRoom = useCallback(
    async (r: AdminRoom) => {
      const name = window.prompt("New room name?", r.name);
      if (name === null) return;
      const capStr = window.prompt("New capacity?", String(r.capacity));
      if (capStr === null) return;
      const capacity = Number(capStr);
      if (!Number.isFinite(capacity) || capacity < 0) {
        setToast({ kind: "error", message: "Invalid capacity" });
        return;
      }
      try {
        await updateAdminRoom(r.id, { name, capacity });
        setToast({ kind: "success", message: "Room updated" });
        await refresh();
      } catch (e) {
        setToast({ kind: "error", message: (e as Error).message });
      }
    },
    [refresh],
  );

  const onDeleteRoom = useCallback(
    async (r: AdminRoom) => {
      const ok = window.confirm(`Soft-delete room "${r.name}"?`);
      if (!ok) return;
      try {
        await deleteAdminRoom(r.id);
        setToast({ kind: "success", message: "Room deleted (soft)" });
        await refresh();
      } catch (e) {
        setToast({ kind: "error", message: (e as Error).message });
      }
    },
    [refresh],
  );

  const banner =
    error && error.includes("Failed to fetch")
      ? "Can’t reach master-service. Check MASTER_API_PROXY_TARGET and that master-service is running."
      : error;

  return (
    <div className="mx-auto max-w-6xl">
      <header className="mb-6 text-center text-white">
        <h1 className="text-3xl font-semibold tracking-tight">Admin</h1>
        <p className="mt-2 text-white/80">Manage customers (entities), users, and rooms in one place.</p>
      </header>

      {toast ? <Toast kind={toast.kind} message={toast.message} onClose={() => setToast(null)} /> : null}

      <section className="mb-6 rounded-xl bg-white p-4 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex flex-wrap gap-2">
            {(["entities", "users", "rooms"] as const).map((t) => (
              <button
                key={t}
                className={`rounded-full px-4 py-1.5 text-sm font-semibold ${
                  tab === t ? "bg-zinc-900 text-white" : "bg-zinc-100 text-zinc-800 hover:bg-zinc-200"
                }`}
                onClick={() => setTab(t)}
              >
                {t.toUpperCase()}
              </button>
            ))}
          </div>
          <div className="text-xs text-zinc-500">
            {loading ? "Refreshing…" : "Ready"}
            {lastUpdatedAt ? <span className="ml-2">Last updated: {lastUpdatedAt}</span> : null}
          </div>
        </div>

        {banner ? <div className="mt-3 text-sm text-red-600">{banner}</div> : null}
      </section>

      {tab === "entities" ? (
        <section className="rounded-xl bg-white p-5 shadow-sm">
          <div className="mb-3 flex items-center justify-between gap-2">
            <div className="text-lg font-semibold text-zinc-900">Entities</div>
            <button
              className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
              onClick={onCreateEntity}
              disabled={loading}
            >
              Add Entity
            </button>
          </div>
          <div className="overflow-auto rounded-lg border border-zinc-200">
            <table className="min-w-[900px] w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs font-semibold tracking-widest text-zinc-500">
                <tr>
                  <th className="px-4 py-3">NAME</th>
                  <th className="px-4 py-3">PHONE</th>
                  <th className="px-4 py-3">EMAIL</th>
                  <th className="px-4 py-3">JOINED</th>
                  <th className="px-4 py-3">ACTIONS</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {entities.map((e) => (
                  <tr key={e.id} className="hover:bg-zinc-50">
                    <td className="px-4 py-3 font-semibold text-zinc-900">{e.name}</td>
                    <td className="px-4 py-3 text-zinc-700">{e.phone}</td>
                    <td className="px-4 py-3 text-zinc-700">{e.email || "-"}</td>
                    <td className="px-4 py-3 text-zinc-700">{new Date(e.joining_date).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          className="rounded bg-zinc-900 px-3 py-1 text-xs font-semibold text-white hover:bg-zinc-800"
                          onClick={() => onEditEntity(e)}
                          disabled={loading}
                        >
                          Edit
                        </button>
                        <button
                          className="rounded bg-red-600 px-3 py-1 text-xs font-semibold text-white hover:bg-red-700"
                          onClick={() => onDeleteEntity(e)}
                          disabled={loading}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {entities.length === 0 ? (
                  <tr>
                    <td className="px-4 py-6 text-sm text-zinc-600" colSpan={5}>
                      No entities.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      {tab === "users" ? (
        <section className="rounded-xl bg-white p-5 shadow-sm">
          <div className="mb-3 flex items-center justify-between gap-2">
            <div className="text-lg font-semibold text-zinc-900">Users</div>
            <button
              className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
              onClick={onCreateUser}
              disabled={loading}
            >
              Add User
            </button>
          </div>
          <div className="overflow-auto rounded-lg border border-zinc-200">
            <table className="min-w-[900px] w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs font-semibold tracking-widest text-zinc-500">
                <tr>
                  <th className="px-4 py-3">USER_ID</th>
                  <th className="px-4 py-3">NAME</th>
                  <th className="px-4 py-3">EMAIL</th>
                  <th className="px-4 py-3">ADMIN</th>
                  <th className="px-4 py-3">CREATED</th>
                  <th className="px-4 py-3">ACTIONS</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {users.map((u) => (
                  <tr key={u.id} className="hover:bg-zinc-50">
                    <td className="px-4 py-3 font-mono text-zinc-800">{u.user_id}</td>
                    <td className="px-4 py-3 font-semibold text-zinc-900">{u.name}</td>
                    <td className="px-4 py-3 text-zinc-700">{u.email}</td>
                    <td className="px-4 py-3">
                      {u.is_admin ? (
                        <span className="rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-800">
                          YES
                        </span>
                      ) : (
                        <span className="rounded-full bg-zinc-100 px-3 py-1 text-xs font-semibold text-zinc-700">
                          NO
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-zinc-700">{new Date(u.created_at).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          className="rounded bg-zinc-900 px-3 py-1 text-xs font-semibold text-white hover:bg-zinc-800"
                          onClick={() => onEditUser(u)}
                          disabled={loading}
                        >
                          Edit
                        </button>
                        <button
                          className="rounded bg-red-600 px-3 py-1 text-xs font-semibold text-white hover:bg-red-700"
                          onClick={() => onDeleteUser(u)}
                          disabled={loading}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {users.length === 0 ? (
                  <tr>
                    <td className="px-4 py-6 text-sm text-zinc-600" colSpan={6}>
                      No users.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      {tab === "rooms" ? (
        <section className="rounded-xl bg-white p-5 shadow-sm">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <div className="text-lg font-semibold text-zinc-900">Rooms</div>
            <div className="flex items-center gap-3">
              <label className="flex items-center gap-2 text-sm text-zinc-700">
                <input
                  type="checkbox"
                  checked={includeDeletedRooms}
                  onChange={(e) => setIncludeDeletedRooms(e.target.checked)}
                />
                Include deleted
              </label>
              <button
                className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
                onClick={onCreateRoom}
                disabled={loading}
              >
                Add Room
              </button>
            </div>
          </div>

          <div className="mb-3 flex flex-wrap gap-2 text-sm">
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Active: <span className="font-semibold">{roomsByStatus.active.length}</span>
            </div>
            <div className="rounded-full bg-zinc-100 px-3 py-1 text-zinc-700">
              Deleted: <span className="font-semibold">{roomsByStatus.deleted.length}</span>
            </div>
          </div>

          <div className="overflow-auto rounded-lg border border-zinc-200">
            <table className="min-w-[1000px] w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs font-semibold tracking-widest text-zinc-500">
                <tr>
                  <th className="px-4 py-3">ID</th>
                  <th className="px-4 py-3">NAME</th>
                  <th className="px-4 py-3">CAPACITY</th>
                  <th className="px-4 py-3">STATUS</th>
                  <th className="px-4 py-3">ACTIONS</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {rooms.map((r) => (
                  <tr key={r.id} className={`hover:bg-zinc-50 ${r.deleted_at ? "opacity-60" : ""}`}>
                    <td className="px-4 py-3 font-mono text-zinc-800">{r.id}</td>
                    <td className="px-4 py-3 font-semibold text-zinc-900">{r.name}</td>
                    <td className="px-4 py-3 text-zinc-700">{r.capacity}</td>
                    <td className="px-4 py-3 text-zinc-700">{r.deleted_at ? "DELETED" : "ACTIVE"}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          className="rounded bg-zinc-900 px-3 py-1 text-xs font-semibold text-white hover:bg-zinc-800 disabled:opacity-50"
                          onClick={() => onEditRoom(r)}
                          disabled={loading || !!r.deleted_at}
                          title={r.deleted_at ? "Cannot edit deleted room" : "Edit"}
                        >
                          Edit
                        </button>
                        <button
                          className="rounded bg-red-600 px-3 py-1 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
                          onClick={() => onDeleteRoom(r)}
                          disabled={loading || !!r.deleted_at}
                          title={r.deleted_at ? "Already deleted" : "Soft delete"}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {rooms.length === 0 ? (
                  <tr>
                    <td className="px-4 py-6 text-sm text-zinc-600" colSpan={5}>
                      No rooms.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      <section className="mt-6 rounded-xl bg-white p-4 text-xs text-zinc-600 shadow-sm">
        <div className="font-semibold text-zinc-900">Notes</div>
        <ul className="mt-2 list-disc pl-5">
          <li>Rooms keep a stable <span className="font-mono">id</span>; renaming updates <span className="font-mono">name</span> only.</li>
          <li>Deleting a room is a soft delete (sets <span className="font-mono">deleted_at</span>).</li>
          <li>If an action fails, you’ll see the backend error message here.</li>
        </ul>
      </section>
    </div>
  );
}

