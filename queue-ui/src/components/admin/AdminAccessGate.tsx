"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { AdminPanel } from "./AdminPanel";

type Props = {
  loggedInEmail: string;
};

export function AdminAccessGate({ loggedInEmail }: Props) {
  const router = useRouter();
  const [adminEmail, setAdminEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [authorized, setAuthorized] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting) return;
    setSubmitting(true);
    try {
      const email = adminEmail.trim();
      const pw = password;
      if (!email || !pw) {
        router.replace("/");
        router.refresh();
        return;
      }

      // IMPORTANT: do NOT use /api/auth/login (it sets a cookie).
      // We call master-service directly and only keep auth in memory for this page view.
      const res = await fetch("/master-api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password: pw }),
        cache: "no-store",
      });
      if (!res.ok) {
        router.replace("/");
        router.refresh();
        return;
      }

      const user = (await res.json()) as { is_admin?: boolean };
      if (user?.is_admin !== true) {
        router.replace("/");
        router.refresh();
        return;
      }

      setAuthorized(true);
    } finally {
      setSubmitting(false);
    }
  }

  if (authorized) {
    return <AdminPanel />;
  }

  return (
    <div className="mx-auto max-w-2xl rounded-xl bg-white p-6 shadow-sm">
      <div className="text-lg font-semibold text-zinc-900">Admin access required</div>
      <div className="mt-2 text-sm text-zinc-700">
        You are signed in as <span className="font-mono">{loggedInEmail}</span>. Enter admin credentials to
        continue.
      </div>

      <form onSubmit={onSubmit} className="mt-5">
        <label className="block text-sm font-semibold text-zinc-800">
          Admin email
          <input
            value={adminEmail}
            onChange={(e) => setAdminEmail(e.target.value)}
            type="email"
            autoComplete="username"
            className="mt-2 w-full rounded-lg border border-zinc-200 px-3 py-2 text-zinc-900 outline-none ring-indigo-500 focus:ring-2"
            required
          />
        </label>

        <label className="mt-4 block text-sm font-semibold text-zinc-800">
          Admin password
          <input
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            type="password"
            autoComplete="current-password"
            className="mt-2 w-full rounded-lg border border-zinc-200 px-3 py-2 text-zinc-900 outline-none ring-indigo-500 focus:ring-2"
            required
          />
        </label>

        <button
          type="submit"
          disabled={submitting}
          className="mt-5 w-full rounded-lg bg-zinc-900 px-4 py-2 font-semibold text-white hover:bg-zinc-800 disabled:opacity-60"
        >
          {submitting ? "Verifying…" : "Continue to Admin"}
        </button>

        <div className="mt-3 text-xs text-zinc-500">
          Note: admin access is not persisted; you will be prompted again next time.
        </div>
      </form>
    </div>
  );
}

