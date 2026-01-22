import { NextResponse } from "next/server";

import { askMetricsQuestion, ensureInitialized, refreshMetrics } from "../../../lib/queueAgentMcp";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

type Body =
  | { action: "refresh" }
  | { action: "query"; question: string };

export async function POST(req: Request) {
  let body: Body;
  try {
    body = (await req.json()) as Body;
  } catch {
    return NextResponse.json({ error: "Invalid JSON body" }, { status: 400 });
  }

  try {
    await ensureInitialized();
  } catch (e) {
    const msg = e instanceof Error ? e.message : "Failed to start queue-agent";
    return NextResponse.json({ error: msg }, { status: 500 });
  }

  if (body.action === "refresh") {
    try {
      // Keep cache fresh server-side (queue-agent stores context in-memory).
      const result = await refreshMetrics();
      return NextResponse.json({ ok: true, result }, { status: 200 });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Refresh failed";
      return NextResponse.json({ error: msg }, { status: 502 });
    }
  }

  if (body.action === "query") {
    const q = (body.question ?? "").trim();
    if (!q) return NextResponse.json({ error: "Missing question" }, { status: 400 });

    try {
      const result = await askMetricsQuestion(q);
      return NextResponse.json({ ok: true, result }, { status: 200 });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Query failed";
      return NextResponse.json({ error: msg }, { status: 502 });
    }
  }

  return NextResponse.json({ error: "Invalid action" }, { status: 400 });
}

