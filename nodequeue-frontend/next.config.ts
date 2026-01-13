import type { NextConfig } from "next";

const API_PROXY_TARGET = process.env.API_PROXY_TARGET || "http://localhost:8080";
const MASTER_API_PROXY_TARGET = process.env.MASTER_API_PROXY_TARGET || "http://localhost:8081";

const nextConfig: NextConfig = {
  // Enables a smaller production image via `.next/standalone` output.
  output: "standalone",
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${API_PROXY_TARGET}/:path*`,
      },
      {
        source: "/master-api/:path*",
        destination: `${MASTER_API_PROXY_TARGET}/:path*`,
      },
    ];
  },
};

export default nextConfig;
