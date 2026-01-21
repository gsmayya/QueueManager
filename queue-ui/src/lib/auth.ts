import "server-only";

import crypto from "crypto";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export const AUTH_COOKIE_NAME = "qm_session";
export const SESSION_TTL_SECONDS = 60 * 60 * 24; // 24h

export type Session = {
  id: string;
  user_id: string;
  name: string;
  email: string;
  exp: number; // unix seconds
};

function getSecret(): string {
  const secret = process.env.AUTH_SECRET || "";
  if (!secret) {
    throw new Error("Missing AUTH_SECRET env var");
  }
  return secret;
}

function base64UrlEncode(buf: Buffer): string {
  return buf
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

function base64UrlDecode(s: string): Buffer {
  const padLen = (4 - (s.length % 4)) % 4;
  const padded = s.replace(/-/g, "+").replace(/_/g, "/") + "=".repeat(padLen);
  return Buffer.from(padded, "base64");
}

function hmacSha256(data: string, secret: string): Buffer {
  return crypto.createHmac("sha256", secret).update(data).digest();
}

function timingSafeEqual(a: Buffer, b: Buffer): boolean {
  if (a.length !== b.length) return false;
  return crypto.timingSafeEqual(a, b);
}

export function createSessionToken(user: Omit<Session, "exp">, nowSeconds?: number): string {
  const now = nowSeconds ?? Math.floor(Date.now() / 1000);
  const payload: Session = { ...user, exp: now + SESSION_TTL_SECONDS };
  const payloadPart = base64UrlEncode(Buffer.from(JSON.stringify(payload), "utf8"));
  const sig = hmacSha256(payloadPart, getSecret());
  const sigPart = base64UrlEncode(sig);
  return `${payloadPart}.${sigPart}`;
}

export function verifySessionToken(token: string): Session | null {
  const parts = token.split(".");
  if (parts.length !== 2) return null;
  const [payloadPart, sigPart] = parts;
  if (!payloadPart || !sigPart) return null;

  const expectedSig = hmacSha256(payloadPart, getSecret());
  let providedSig: Buffer;
  try {
    providedSig = base64UrlDecode(sigPart);
  } catch {
    return null;
  }
  if (!timingSafeEqual(expectedSig, providedSig)) return null;

  let payload: Session;
  try {
    payload = JSON.parse(base64UrlDecode(payloadPart).toString("utf8")) as Session;
  } catch {
    return null;
  }
  if (!payload?.exp || typeof payload.exp !== "number") return null;
  const now = Math.floor(Date.now() / 1000);
  if (payload.exp <= now) return null;
  if (!payload.id || !payload.email || !payload.user_id) return null;
  return payload;
}

export async function getSession(): Promise<Session | null> {
  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_COOKIE_NAME)?.value;
  if (!token) return null;
  return verifySessionToken(token);
}

export async function requireSession(nextPath: string): Promise<Session> {
  const session = await getSession();
  if (!session) {
    redirect(`/login?next=${encodeURIComponent(nextPath)}`);
  }
  return session;
}

