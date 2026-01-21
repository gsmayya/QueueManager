import Link from "next/link";
import { redirect } from "next/navigation";

import { getSession } from "../../lib/auth";
import { LoginForm } from "../../components/auth/LoginForm";

export default async function LoginPage({
  searchParams,
}: {
  searchParams?: { next?: string };
}) {
  const session = await getSession();
  const next = (searchParams?.next || "/").toString();
  if (session) {
    redirect(next);
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-indigo-600 via-purple-700 to-fuchsia-700 px-5 py-10">
      <div className="mx-auto flex max-w-5xl flex-col items-center gap-6">
        <div className="text-center text-white">
          <h1 className="text-3xl font-semibold tracking-tight">Login</h1>
          <p className="mt-2 text-white/80">Sign in to access Admin, Queue, Nodes, and Metrics.</p>
          <div className="mt-3">
            <Link href="/" className="text-sm text-white/80 hover:text-white">
              ← Back to home
            </Link>
          </div>
        </div>

        <LoginForm />
      </div>
    </div>
  );
}

