import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "TokenHub Admin",
  description: "Enterprise AI Gateway administration console",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className="antialiased">{children}</body>
    </html>
  );
}

