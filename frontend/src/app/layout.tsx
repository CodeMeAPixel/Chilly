import type { Metadata, Viewport } from "next";
import { Fredoka, Geist, Geist_Mono } from "next/font/google";
import { Providers } from "./providers";
import "./globals.css";

const sans = Geist({ variable: "--font-sans", subsets: ["latin"] });
const mono = Geist_Mono({ variable: "--font-mono", subsets: ["latin"] });
const display = Fredoka({ variable: "--font-display", subsets: ["latin"], weight: ["500", "600", "700"] });

const siteUrl = process.env.SITE_URL ?? "http://localhost:3000";
const description =
  "Chilly is a laid-back Discord music bot. Play from YouTube, SoundCloud and Spotify, run 24/7 radio, save playlists and control everything from a live web dashboard.";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Chilly — chill music for your Discord server",
    template: "%s · Chilly",
  },
  description,
  applicationName: "Chilly",
  keywords: ["discord music bot", "discord bot", "lavalink", "music", "radio", "playlists"],
  openGraph: {
    type: "website",
    siteName: "Chilly",
    title: "Chilly — chill music for your Discord server",
    description,
    url: "/",
  },
  twitter: {
    card: "summary_large_image",
    title: "Chilly — chill music for your Discord server",
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
