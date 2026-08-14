import type { Metadata } from "next";
import { headers } from "next/headers";
import "./globals.css";

export async function generateMetadata(): Promise<Metadata> {
  const requestHeaders = await headers();
  const host =
    requestHeaders.get("x-forwarded-host") ??
    requestHeaders.get("host") ??
    "127.0.0.1:5658";
  const protocol =
    requestHeaders.get("x-forwarded-proto") ??
    (host.startsWith("127.") || host.startsWith("localhost") ? "http" : "https");
  const socialImage = `${protocol}://${host}/og.png`;

  return {
    title: "Ray Film Lab · 3D Tensor Viewer",
    description: "在浏览器本地解析 Ray film 二进制文件，并用 WebGL2 检查三维张量体与正交切片。",
    icons: {
      icon: "/favicon.svg",
      shortcut: "/favicon.svg",
    },
    openGraph: {
      title: "Ray Film Lab",
      description: "Local 3D tensor inspection for Ray film files.",
      type: "website",
      images: [socialImage],
    },
    twitter: {
      card: "summary_large_image",
      title: "Ray Film Lab",
      description: "Local 3D tensor inspection for Ray film files.",
      images: [socialImage],
    },
  };
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}
