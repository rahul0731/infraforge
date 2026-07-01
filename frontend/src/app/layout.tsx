import "./globals.css";
import type { Metadata } from "next";
import Sidebar from "@/components/Sidebar";

export const metadata: Metadata = {
  title: "InfraForge",
  description: "Platform Engineering Dashboard",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <body className="flex min-h-screen bg-dark-bg">
        <Sidebar />
        <main className="flex-1 ml-60 p-6 overflow-auto min-h-screen">{children}</main>
      </body>
    </html>
  );
}
