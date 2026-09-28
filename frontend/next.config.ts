import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  compress: false,
  poweredByHeader: false,
};

export default nextConfig;
