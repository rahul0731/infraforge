"use client";
import { useEffect, useState } from "react";
import { fetchDashboardStats, fetchEnvironments, fetchTeams } from "@/lib/api";
import StatCard from "@/components/StatCard";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null);
  const [environments, setEnvironments] = useState<any[]>([]);
  const [teamId, setTeamId] = useState<string>("");

  useEffect(() => {
    // Get first team and use it
    fetchTeams().then((teams) => {
      if (teams && teams.length > 0) {
        setTeamId(teams[0].id);
      }
    });
  }, []);

  useEffect(() => {
    if (!teamId) return;
    const load = () => {
      fetchDashboardStats(teamId).then(setStats).catch(console.error);
      fetchEnvironments(teamId).then(setEnvironments).catch(console.error);
    };
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, [teamId]);

  if (!stats) {
    return <div className="text-dark-muted">Loading dashboard...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <Link
          href="/environments/new"
          className="px-4 py-2 bg-dark-accent text-white rounded-lg text-sm hover:bg-indigo-600 transition-colors"
        >
          + New Environment
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard label="Environments" value={stats.total_environments} icon="🌍" />
        <StatCard label="Active Workflows" value={stats.active_workflows} icon="⚙️" color="text-blue-400" />
        <StatCard label="Pending Approvals" value={stats.pending_approvals} icon="⏳" color="text-yellow-400" />
        <StatCard label="Unresolved Drifts" value={stats.unresolved_drifts} icon="🔀" color="text-red-400" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <StatCard label="Completed Workflows" value={stats.completed_workflows} icon="✅" color="text-emerald-400" />
        <StatCard label="Failed Workflows" value={stats.failed_workflows} icon="❌" color="text-red-400" />
        <StatCard label="Critical Drifts" value={stats.critical_drifts} icon="🚨" color="text-red-500" />
      </div>

      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <h2 className="text-lg font-semibold mb-4">Environments</h2>
        {environments.length === 0 ? (
          <p className="text-dark-muted text-sm">No environments yet. Create one to get started.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-dark-muted border-b border-dark-border">
                  <th className="text-left py-2 px-3">Name</th>
                  <th className="text-left py-2 px-3">Provider</th>
                  <th className="text-left py-2 px-3">Region</th>
                  <th className="text-left py-2 px-3">Status</th>
                  <th className="text-left py-2 px-3">Created</th>
                </tr>
              </thead>
              <tbody>
                {environments.map((env) => (
                  <tr key={env.id} className="border-b border-dark-border/50 hover:bg-dark-border/20">
                    <td className="py-3 px-3">
                      <Link href={`/environments/${env.id}`} className="text-dark-accent hover:underline">
                        {env.name}
                      </Link>
                    </td>
                    <td className="py-3 px-3 text-dark-muted">{env.provider}</td>
                    <td className="py-3 px-3 text-dark-muted">{env.region || "—"}</td>
                    <td className="py-3 px-3"><StatusBadge status={env.status} /></td>
                    <td className="py-3 px-3 text-dark-muted">{new Date(env.created_at).toLocaleDateString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
