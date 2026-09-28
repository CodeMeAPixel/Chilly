import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Chilly",
    short_name: "Chilly",
    description: "Chill music for your Discord server.",
    start_url: "/dashboard",
    display: "standalone",
    background_color: "#0a1014",
    theme_color: "#0a1014",
    icons: [
      { src: "/icon.svg", sizes: "any", type: "image/svg+xml" },
      { src: "/apple-icon", sizes: "180x180", type: "image/png" },
    ],
  };
}
