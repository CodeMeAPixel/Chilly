import type { Metadata, Viewport } from "next";
import { Fredoka, Geist, Geist_Mono } from "next/font/google";
import { Providers } from "./providers";
import "./globals.css";

const sans = Geist({ variable: "--font-sans", subsets: ["latin"] });
const mono = Geist_Mono({ variable: "--font-mono", subsets: ["latin"] });
const display = Fredoka({ variable: "--font-display", subsets: ["latin"], weight: ["500", "600", "700"] });

const siteUrl = process.env.SITE_URL ?? "http://localhost:3000";
const description =
  "Chilly is a 24/7 radio bot for Discord. Tune your voice channel into always-on stations, keep them playing around the clock, play any song on request and control it all from a live web dashboard.";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Chilly — 24/7 radio for your Discord server",
    template: "%s · Chilly",
  },
  description,
  applicationName: "Chilly",
  keywords: ["discord radio bot", "24/7 discord radio", "discord music bot", "lofi discord bot", "radio", "music"],
  openGraph: {
    type: "website",
    siteName: "Chilly",
    title: "Chilly — 24/7 radio for your Discord server",
    description,
    url: "/",
  },
  twitter: {
    card: "summary_large_image",
    title: "Chilly — 24/7 radio for your Discord server",
    description,
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: dark)", color: "#0a1014" },
    { media: "(prefers-color-scheme: light)", color: "#f6fbfa" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${sans.variable} ${mono.variable} ${display.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col font-sans">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
