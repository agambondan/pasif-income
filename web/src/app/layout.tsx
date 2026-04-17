import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Pasif Income Dashboard",
  description: "Dashboard for faceless content generation and clip tracking",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
