import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  turbopack: { root: __dirname },
  outputFileTracingRoot: __dirname,
  // Let phones on the same Wi-Fi load the dev server (Next blocks /_next/* from other origins by default).
  // Update the IP if the Mac gets a new DHCP address (`ipconfig getifaddr en0`).
  allowedDevOrigins: ["192.168.1.4", "localhost", "*.local"],
  // No floating "N" dev button over the game HUD.
  devIndicators: false,
};

export default nextConfig;
