import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";

const size = 512;

const logo = readFile(join(process.cwd(), "public", "logo.svg")).then(
  (svg) => `data:image/svg+xml;base64,${svg.toString("base64")}`,
);

export async function GET() {
  const src = await logo;
  return new ImageResponse(
    (
      <div style={{ width: "100%", height: "100%", display: "flex", alignItems: "center", justifyContent: "center" }}>
        <img src={src} width={size} height={size} />
      </div>
    ),
    {
      width: size,
      height: size,
      headers: {
        "Content-Disposition": 'inline; filename="chilly-logo.png"',
        "Cache-Control": "public, max-age=86400, immutable",
      },
    },
  );
}
