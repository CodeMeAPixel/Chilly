import { ogSize, renderOgImage } from "@/lib/og";

export const alt = "Chilly Radio — always on air";
export const size = ogSize;
export const contentType = "image/png";

export default function Image() {
  return renderOgImage({
    eyebrow: "Chilly Radio",
    title: "Always on air",
    subtitle: "Tune in from your browser or send a station to your voice channel.",
  });
}
