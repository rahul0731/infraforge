"use client";
import { useEffect, useState } from "react";
import { fetchWorkflows, fetchTeams, retryWorkflow } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";

export default function WorkflowsPage() {
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [teamId, setTeamId] = useState("");
  const [filter, setFilter] = useState("");
  const [retryingId, setRetryingId] = useState("");

  useEffect(() => {
    fetchTeams().then((t) => { if (t?.length) setTeamId(t[0].id); });
  }, []);

  useEffect(() => {
    if (!teamId) return;
    const load = () => fetchWorkflows(teamId, filter).then((data) => setWorkflows(data || [])).catch(console.error);
    load();
    const interval = setInterval(load, 3000);
    return () => clearInterval(interval);
  }, [teamId, filter]);

  const handleRetry = async (e: React.MouseEvent, wfId: string) => {
    e.preventDefault();
    e.stopPropagation();
    setRetryingId(wfId);
    try { await retryWorkflow(wfId); } catch (err) { console.error(err); }
    setRetryingId("");
  };

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
          <div key={wf.id} className="bg-dark-card border border-dark-border rounded-xl p-4 hover:border-dark-accent/50 transition-colors">
            <div className="flex items-center justify-between">
              <Link href={`/workflows/${wf.id}`} className="flex-1">
                <p className="font-medium">{wf.name}</p>
                <p className="text-sm text-dark-muted mt-1">
                  {wf.workflow_type} · by {wf.initiated_by} · {new Date(wf.created_at).toLocaleString()}
                </p>
              </Link>
              <div className="flex items-center gap-3">
                {(wf.status === "failed" || wf.status === "cancelled") && (
                  <button
                    onClick={(e) => handleRetry(e, wf.id)}
                    disabled={retryingId === wf.id}
                    className="px-3 py-1.5 bg-dark-accent text-white rounded-lg text-xs hover:bg-indigo-600 disabled:opacity-50"
                  >
                    {retryingId === wf.id ? "..." : "Retry"}
                  </button>
                )}
                <StatusBadge status={wf.status} />
              </div>
            </div>
          </div>
        ))}
        {workflows.length === 0 && (
          <p className="text-dark-muted text-center py-10">No workflows found.</p>
        )}
      </div>
    </div>
  );
}
