import type { MetadataRoute } from "next";

export const dynamic = "force-dynamic";

export default function sitemap(): MetadataRoute.Sitemap {
  const siteUrl = process.env.SITE_URL ?? "http://localhost:3000";
  return [
    { url: `${siteUrl}/`, changeFrequency: "weekly", priority: 1 },
    { url: `${siteUrl}/radio`, changeFrequency: "daily", priority: 0.8 },
    { url: `${siteUrl}/status`, changeFrequency: "always", priority: 0.5 },
  ];
}
