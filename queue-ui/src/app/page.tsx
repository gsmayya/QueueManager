import Link from "next/link";
import { getSession } from "../lib/auth";

export default async function Home() {
  const session = await getSession();
  return (
    <div className="min-h-screen bg-gradient-to-br from-indigo-600 via-purple-700 to-fuchsia-700 px-5 py-10">
      <div className="mx-auto max-w-5xl text-white">
        <header className="text-center">
          <h1 className="text-4xl font-semibold tracking-tight">Queue Manager</h1>
          <p className="mt-3 text-white/80">
            Quick links to monitoring, administration, and live waiting rooms.
          </p>
        </header>

        <section className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-2">
          <Link
            href="/rooms/waiting"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">PUBLIC</div>
            <div className="mt-2 text-2xl font-semibold">Waiting Rooms</div>
            <div className="mt-2 text-white/80">
              View live service + waiting queues by room. No login required.
            </div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>

          <Link
            href="/agent"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">PUBLIC</div>
            <div className="mt-2 text-2xl font-semibold">Agent</div>
            <div className="mt-2 text-white/80">
              Ask questions about node + resource metrics using the queue-agent.
            </div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>

          <Link
            href="/admin"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">LOGIN REQUIRED</div>
            <div className="mt-2 text-2xl font-semibold">Admin</div>
            <div className="mt-2 text-white/80">Manage users, entities, and rooms.</div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>

          <Link
            href="/node"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">LOGIN REQUIRED</div>
            <div className="mt-2 text-2xl font-semibold">Nodes</div>
            <div className="mt-2 text-white/80">Inspect and interact with active nodes.</div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>

          <Link
            href="/metrics"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">LOGIN REQUIRED</div>
            <div className="mt-2 text-2xl font-semibold">Metrics</div>
            <div className="mt-2 text-white/80">Session metrics for nodes and resources.</div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>

          <Link
            href="/schedules"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">LOGIN REQUIRED</div>
            <div className="mt-2 text-2xl font-semibold">Schedules</div>
            <div className="mt-2 text-white/80">Create and manage recurring schedules with time limits.</div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>

          <Link
            href="/queue"
            className="group rounded-2xl bg-white/10 p-6 shadow-sm ring-1 ring-white/10 transition hover:bg-white/15"
          >
            <div className="text-sm font-semibold tracking-widest text-white/70">LOGIN REQUIRED</div>
            <div className="mt-2 text-2xl font-semibold">Manage Queue</div>
            <div className="mt-2 text-white/80">
              Allocate, move, and complete nodes across resources.
            </div>
            <div className="mt-5 text-sm font-semibold text-white/90 group-hover:text-white">
              Open →
            </div>
          </Link>
        </section>

        <div className="mt-10 flex flex-wrap items-center justify-center gap-3 text-sm">
          {session ? (
            <form action="/api/auth/logout" method="post">
              <button
                type="submit"
                className="rounded-full bg-white px-4 py-2 font-semibold text-zinc-900 hover:bg-white/90"
              >
                Logout
              </button>
            </form>
          ) : (
            <Link
              href="/login"
              className="rounded-full bg-white px-4 py-2 font-semibold text-zinc-900 hover:bg-white/90"
            >
              Login
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}
