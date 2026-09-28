import { ogSize, renderOgImage } from "@/lib/og";

export const alt = "Chilly status";
export const size = ogSize;
export const contentType = "image/png";

export default function Image() {
  return renderOgImage({
    eyebrow: "System status",
    title: "Is Chilly up?",
    subtitle: "Live health for the Discord connection, audio nodes and radio.",
  });
}
