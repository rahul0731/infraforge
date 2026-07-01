const statusColors: Record<string, string> = {
  active: "bg-emerald-500/20 text-emerald-400",
  completed: "bg-emerald-500/20 text-emerald-400",
  approved: "bg-emerald-500/20 text-emerald-400",
  running: "bg-blue-500/20 text-blue-400",
  pending: "bg-yellow-500/20 text-yellow-400",
  failed: "bg-red-500/20 text-red-400",
  cancelled: "bg-gray-500/20 text-gray-400",
  rejected: "bg-red-500/20 text-red-400",
  decommissioned: "bg-gray-500/20 text-gray-400",
  skipped: "bg-gray-500/20 text-gray-400",
  critical: "bg-red-500/20 text-red-400",
  high: "bg-orange-500/20 text-orange-400",
  medium: "bg-yellow-500/20 text-yellow-400",
  low: "bg-blue-500/20 text-blue-400",
};

export default function StatusBadge({ status }: { status: string }) {
  const color = statusColors[status] || "bg-gray-500/20 text-gray-400";
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${color}`}>
      {status}
    </span>
  );
}
