import { ogSize, renderOgImage } from "@/lib/og";

export const alt = "Chilly — 24/7 radio for your Discord server";
export const size = ogSize;
export const contentType = "image/png";

export default function Image() {
  return renderOgImage({
    eyebrow: "Discord radio bot",
    title: "24/7 radio for your Discord server",
    subtitle: "Always-on stations, songs on request and a live web dashboard.",
  });
}
