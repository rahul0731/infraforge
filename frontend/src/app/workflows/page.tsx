"use client";
import { useEffect, useState } from "react";
import { fetchWorkflows, fetchTeams } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";

export default function WorkflowsPage() {
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [teamId, setTeamId] = useState("");
  const [filter, setFilter] = useState("");

  useEffect(() => {
    fetchTeams().then((t) => { if (t?.length) setTeamId(t[0].id); });
  }, []);

  useEffect(() => {
    if (!teamId) return;
    const load = () => fetchWorkflows(teamId, filter).then(setWorkflows).catch(console.error);
    load();
    const interval = setInterval(load, 3000);
    return () => clearInterval(interval);
  }, [teamId, filter]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Workflows</h1>
        <select
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="bg-dark-card border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
        >
          <option value="">All statuses</option>
          <option value="running">Running</option>
          <option value="pending">Pending</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      <div className="space-y-3">
        {workflows.map((wf) => (
          <Link
            key={wf.id}
            href={`/workflows/${wf.id}`}
            className="block bg-dark-card border border-dark-border rounded-xl p-4 hover:border-dark-accent/50 transition-colors"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">{wf.name}</p>
                <p className="text-sm text-dark-muted mt-1">
                  {wf.workflow_type} · by {wf.initiated_by} · {new Date(wf.created_at).toLocaleString()}
                </p>
              </div>
              <StatusBadge status={wf.status} />
            </div>
          </Link>
        ))}
        {workflows.length === 0 && (
          <p className="text-dark-muted text-center py-10">No workflows found.</p>
        )}
      </div>
    </div>
  );
}
