"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";

const navItems = [
  { href: "/", label: "Dashboard" },
  { href: "/environments", label: "Environments" },
  { href: "/workflows", label: "Workflows" },
  { href: "/approvals", label: "Approvals" },
  { href: "/drift", label: "Drift" },
];

export default function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed top-0 left-0 w-60 h-screen bg-dark-card border-r border-dark-border flex flex-col z-50">
      <div className="p-6 border-b border-dark-border">
        <h1 className="text-xl font-bold text-dark-accent">InfraForge</h1>
        <p className="text-xs text-dark-muted mt-1">Platform Engineering</p>
      </div>
      <nav className="flex-1 p-4 space-y-1">
        {navItems.map((item) => {
          const isActive = pathname === item.href ||
            (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`block px-3 py-2 rounded-lg text-sm transition-colors ${
                isActive
                  ? "bg-dark-accent/20 text-dark-accent font-medium"
                  : "text-dark-muted hover:text-dark-text hover:bg-dark-border/50"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="p-4 border-t border-dark-border">
        <p className="text-xs text-dark-muted">Team: Platform</p>
      </div>
    </aside>
  );
}
