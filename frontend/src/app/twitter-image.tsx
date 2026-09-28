import { ogSize, renderOgImage } from "@/lib/og";

export const alt = "Chilly — chill music for your Discord server";
export const size = ogSize;
export const contentType = "image/png";

export default function Image() {
  return renderOgImage({
    eyebrow: "Discord music bot",
    title: "Chill music for your Discord server",
    subtitle: "Playlists, 24/7 radio and a live web dashboard.",
  });
}
