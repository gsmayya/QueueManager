import Link from "next/link";
import { requireSession } from "../../lib/auth";
import { SchedulesPage } from "../../components/schedules/SchedulesPage";

export default async function Page() {
  await requireSession("/schedules");
  return (
    <div className="min-h-screen bg-gradient-to-br from-indigo-500 to-purple-700 px-5 py-8">
      <div className="mx-auto mb-4 max-w-6xl">
        <Link href="/" className="text-sm text-white/80 hover:text-white">
          ← Back to main
        </Link>
      </div>
      <SchedulesPage />
    </div>
  );
}

