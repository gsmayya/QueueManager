import Link from "next/link";
import { requireSession } from "../../lib/auth";
import { AdminAccessGate } from "../../components/admin/AdminAccessGate";

export default async function AdminPage() {
  const session = await requireSession("/admin");
  return (
    <div className="min-h-screen bg-gradient-to-br from-zinc-900 to-zinc-700 px-5 py-8">
      <div className="mx-auto mb-4 max-w-6xl">
        <Link href="/" className="text-sm text-white/80 hover:text-white">
          ← Back to main
        </Link>
      </div>
      <AdminAccessGate loggedInEmail={session.email} />
    </div>
  );
}

