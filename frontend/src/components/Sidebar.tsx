"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";

const navItems = [
  { href: "/", label: "Dashboard", icon: "📊" },
  { href: "/environments", label: "Environments", icon: "🌍" },
  { href: "/workflows", label: "Workflows", icon: "⚙️" },
  { href: "/approvals", label: "Approvals", icon: "✅" },
  { href: "/drift", label: "Drift", icon: "🔀" },
];

export default function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-dark-card border-r border-dark-border flex flex-col">
      <div className="p-6 border-b border-dark-border">
        <h1 className="text-xl font-bold text-dark-accent">⚒️ InfraForge</h1>
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
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                isActive
                  ? "bg-dark-accent/20 text-dark-accent"
                  : "text-dark-muted hover:text-dark-text hover:bg-dark-border/50"
              }`}
            >
              <span>{item.icon}</span>
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>
      <div className="p-4 border-t border-dark-border">
        <div className="text-xs text-dark-muted">Team: Platform</div>
      </div>
    </aside>
  );
}
