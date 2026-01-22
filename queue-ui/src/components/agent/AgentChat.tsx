"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

type ChatMsg = {
  id: string;
  role: "user" | "agent";
  text: string;
  context?: string;
};

type ApiOk = { ok: true; result?: unknown };
type ApiErr = { error: string };

function getErrorMessage(v: unknown): string | null {
  if (v && typeof v === "object" && "error" in v && typeof (v as { error?: unknown }).error === "string") {
    return (v as { error: string }).error;
  }
  return null;
}

async function postAgent(body: unknown) {
  const res = await fetch("/api/agent", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const json: unknown = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(getErrorMessage(json) || res.statusText);
  }
  return json as ApiOk | ApiErr;
}

export function AgentChat() {
  const [messages, setMessages] = useState<ChatMsg[]>([]);
  const [question, setQuestion] = useState("");
  const [busy, setBusy] = useState(false);
  const canSend = useMemo(() => !busy && question.trim().length > 0, [busy, question]);
  const didInitialRefresh = useRef(false);

  const refresh = useCallback(async (opts?: { silent?: boolean }) => {
    setBusy(true);
    try {
      await postAgent({ action: "refresh" });
      if (!opts?.silent) {
        setMessages((m) => [
          ...m,
          { id: crypto.randomUUID(), role: "agent", text: "Refreshed metrics context from queue-service." },
        ]);
      }
    } catch (e) {
      setMessages((m) => [
        ...m,
        { id: crypto.randomUUID(), role: "agent", text: `Refresh failed: ${e instanceof Error ? e.message : "unknown error"}` },
      ]);
    } finally {
      setBusy(false);
    }
  }, []);

  // Refresh context as soon as the agent page loads.
  useEffect(() => {
    if (didInitialRefresh.current) return;
    didInitialRefresh.current = true;
    void refresh({ silent: true });
  }, [refresh]);

  const send = useCallback(async () => {
    const q = question.trim();
    if (!q) return;
    setQuestion("");
    setBusy(true);
    const userMsg: ChatMsg = { id: crypto.randomUUID(), role: "user", text: q };
    setMessages((m) => [...m, userMsg]);
    try {
      const json = await postAgent({ action: "query", question: q });
      const result = (json as ApiOk).result;
      const answer =
        result && typeof result === "object" && "answer" in result && typeof (result as { answer?: unknown }).answer === "string"
          ? (result as { answer: string }).answer
          : "(no answer)";
      const context =
        result && typeof result === "object" && "context" in result && typeof (result as { context?: unknown }).context === "string"
          ? (result as { context: string }).context
          : undefined;
      setMessages((m) => [...m, { id: crypto.randomUUID(), role: "agent", text: answer, context }]);
    } catch (e) {
      setMessages((m) => [
        ...m,
        { id: crypto.randomUUID(), role: "agent", text: `Query failed: ${e instanceof Error ? e.message : "unknown error"}` },
      ]);
    } finally {
      setBusy(false);
    }
  }, [question]);

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">Agent</h1>
          <p className="text-sm opacity-80">Ask questions about node/resource metrics (powered by `queue-agent`).</p>
        </div>
        <button
          type="button"
          onClick={refresh}
          disabled={busy}
          className="rounded-md border px-3 py-2 text-sm hover:opacity-90 disabled:opacity-50"
        >
          Refresh context
        </button>
      </div>

      <div className="rounded-lg border p-4">
        {messages.length === 0 ? (
          <div className="text-sm opacity-70">
            Try: <span className="font-mono">Which resource has the highest avg waiting time?</span>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {messages.map((m) => (
              <div key={m.id} className="flex flex-col gap-2">
                <div className="text-xs uppercase tracking-wide opacity-60">{m.role}</div>
                <pre className="whitespace-pre-wrap rounded-md bg-black/5 p-3 text-sm dark:bg-white/5">{m.text}</pre>
                {m.context ? (
                  <details className="rounded-md border p-3 text-sm">
                    <summary className="cursor-pointer select-none">Raw context</summary>
                    <pre className="mt-2 whitespace-pre-wrap text-xs opacity-90">{m.context}</pre>
                  </details>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </div>

      <form
        className="flex flex-col gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          void send();
        }}
      >
        <textarea
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="Ask a question…"
          rows={3}
          className="w-full resize-y rounded-md border p-3 text-sm"
        />
        <div className="flex items-center justify-end gap-2">
          <button
            type="submit"
            disabled={!canSend}
            className="rounded-md bg-black px-4 py-2 text-sm text-white hover:opacity-90 disabled:opacity-50 dark:bg-white dark:text-black"
          >
            {busy ? "Working…" : "Send"}
          </button>
        </div>
      </form>
    </div>
  );
}

