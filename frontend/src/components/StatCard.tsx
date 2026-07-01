interface StatCardProps {
  label: string;
  value: number | string;
  color?: string;
}

export default function StatCard({ label, value, color = "text-dark-accent" }: StatCardProps) {
  return (
    <div className="bg-dark-card border border-dark-border rounded-xl p-5">
      <p className="text-sm text-dark-muted">{label}</p>
      <p className={`text-2xl font-bold mt-1 ${color}`}>{value}</p>
    </div>
  );
}
