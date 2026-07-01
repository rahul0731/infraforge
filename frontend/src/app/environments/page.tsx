"use client";
import { useEffect, useState } from "react";
import { fetchEnvironments, fetchTeams } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";

export default function EnvironmentsPage() {
  const [environments, setEnvironments] = useState<any[]>([]);
  const [teamId, setTeamId] = useState<string>("");

  useEffect(() => {
    fetchTeams().then((teams) => {
      if (teams?.length) setTeamId(teams[0].id);
    });
  }, []);

  useEffect(() => {
    if (!teamId) return;
    const load = () => fetchEnvironments(teamId).then((data) => setEnvironments(data || [])).catch(console.error);
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, [teamId]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Environments</h1>
        <Link
          href="/environments/new"
          className="px-4 py-2 bg-dark-accent text-white rounded-lg text-sm hover:bg-indigo-600 transition-colors"
        >
          + New Environment
        </Link>
      </div>

      <div className="grid gap-4">
        {environments.map((env) => (
          <div key={env.id} className="bg-dark-card border border-dark-border rounded-xl p-5 flex items-center justify-between">
            <div>
              <Link href={`/environments/${env.id}`} className="text-lg font-medium text-dark-accent hover:underline">
                {env.name}
              </Link>
              <p className="text-sm text-dark-muted mt-1">
                {env.provider.toUpperCase()} · {env.region || "no region"} · {env.slug}
              </p>
            </div>
            <div className="flex items-center gap-4">
              <StatusBadge status={env.status} />
              <Link href={`/environments/${env.id}/edit`} className="text-sm text-dark-muted hover:text-dark-text">
                Edit
              </Link>
            </div>
          </div>
        ))}
        {environments.length === 0 && (
          <p className="text-dark-muted text-center py-10">No environments found.</p>
        )}
      </div>
    </div>
  );
}
