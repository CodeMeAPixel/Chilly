import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";

export const ogSize = { width: 1200, height: 630 };

const assets = Promise.all([
  readFile(join(process.cwd(), "public", "logo.svg")),
  readFile(join(process.cwd(), "src", "assets", "fonts", "Fredoka-SemiBold.ttf")),
  readFile(join(process.cwd(), "src", "assets", "fonts", "Geist-Regular.ttf")),
]).then(([logo, display, body]) => ({
  logo: `data:image/svg+xml;base64,${logo.toString("base64")}`,
  display,
  body,
}));

export async function renderOgImage({ eyebrow, title, subtitle }: { eyebrow: string; title: string; subtitle: string }) {
  const { logo, display, body } = await assets;

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "72px 88px",
          color: "#e6f3f1",
          fontFamily: "Geist",
          backgroundColor: "#0a1014",
          backgroundImage:
            "radial-gradient(circle at 62% -10%, rgba(181, 235, 229, 0.22), rgba(181, 235, 229, 0) 55%), radial-gradient(circle at 95% 110%, rgba(255, 154, 162, 0.2), rgba(255, 154, 162, 0) 45%)",
        }}
      >
        <div style={{ display: "flex", flexDirection: "column", maxWidth: 700, gap: 24 }}>
          <div
            style={{
              display: "flex",
              alignSelf: "flex-start",
              padding: "8px 22px",
              borderRadius: 9999,
              background: "rgba(181, 235, 229, 0.14)",
              color: "#b5ebe5",
              fontSize: 26,
            }}
          >
            {eyebrow}
          </div>
          <div style={{ fontFamily: "Fredoka", fontSize: 80, lineHeight: 1.02, letterSpacing: -1 }}>{title}</div>
          <div style={{ fontSize: 32, color: "#8da6a3", lineHeight: 1.35 }}>{subtitle}</div>
        </div>
        <img src={logo} width={340} height={340} style={{ transform: "rotate(-8deg)" }} />
      </div>
    ),
    {
      ...ogSize,
      fonts: [
        { name: "Fredoka", data: display, weight: 600, style: "normal" },
        { name: "Geist", data: body, weight: 400, style: "normal" },
      ],
    },
  );
}
