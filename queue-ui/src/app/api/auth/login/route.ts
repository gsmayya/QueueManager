import { NextResponse } from "next/server";

import { AUTH_COOKIE_NAME, createSessionToken } from "../../../../lib/auth";

type LoginBody = {
  email?: string;
  password?: string;
};

export async function POST(req: Request) {
  let body: LoginBody;
  try {
    body = (await req.json()) as LoginBody;
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }

  const email = (body.email || "").trim();
  const password = (body.password || "").trim();
  if (!email || !password) {
    return NextResponse.json({ error: "email and password are required" }, { status: 400 });
  }

  // Call master-service via Next rewrites (/master-api -> MASTER_API_PROXY_TARGET).
  const url = new URL("/master-api/auth/login", req.url);
  const upstream = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
    cache: "no-store",
  });

  if (!upstream.ok) {
    let errMsg = "Login failed";
    try {
      const j = (await upstream.json()) as { error?: string };
      if (j?.error) errMsg = j.error;
    } catch {
      // ignore
    }
    const status = upstream.status === 401 ? 401 : upstream.status;
    return NextResponse.json({ error: errMsg }, { status });
  }

  const user = (await upstream.json()) as {
    id: string;
    user_id: string;
    name: string;
    email: string;
  };

  const token = createSessionToken({
    id: user.id,
    user_id: user.user_id,
    name: user.name,
    email: user.email,
  });

  const res = NextResponse.json({ user }, { status: 200 });
  res.cookies.set(AUTH_COOKIE_NAME, token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 24,
  });
  return res;
}

