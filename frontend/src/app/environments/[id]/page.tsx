"use client";
import { useEffect, useState } from "react";
import { fetchEnvironment, fetchWorkflows, fetchTeams, fetchDriftRecords } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";
import { useParams } from "next/navigation";

export default function EnvironmentDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [env, setEnv] = useState<any>(null);
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [driftCount, setDriftCount] = useState(0);

  useEffect(() => {
    const load = () => {
      fetchEnvironment(id).then(setEnv).catch(console.error);
      fetchDriftRecords(id, true).then((r) => setDriftCount(r?.length || 0)).catch(() => {});
    };
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, [id]);

  useEffect(() => {
    if (!env) return;
    fetchTeams().then((teams) => {
      if (teams?.length) {
        fetchWorkflows(env.team_id).then((wfs) => {
          setWorkflows(wfs?.filter((w: any) => w.environment_id === id) || []);
        });
      }
    });
  }, [env, id]);

  if (!env) return <div className="text-dark-muted">Loading...</div>;

  const config = typeof env.config === "object" ? env.config : {};

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{env.name}</h1>
          <p className="text-dark-muted text-sm">{env.slug} · {env.provider.toUpperCase()} · {env.region}</p>
        </div>
        <div className="flex gap-3">
          <Link
            href={`/environments/${id}/edit`}
            className="px-4 py-2 bg-dark-border text-dark-text rounded-lg text-sm hover:bg-dark-border/80"
          >
            Edit
          </Link>
          <StatusBadge status={env.status} />
        </div>
      </div>

      {/* Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-dark-card border border-dark-border rounded-xl p-5">
          <p className="text-sm text-dark-muted">Provider</p>
          <p className="text-lg font-medium mt-1">{env.provider.toUpperCase()}</p>
        </div>
        <div className="bg-dark-card border border-dark-border rounded-xl p-5">
          <p className="text-sm text-dark-muted">Instance Size</p>
          <p className="text-lg font-medium mt-1">{config.instance_size || "—"}</p>
        </div>
        <div className="bg-dark-card border border-dark-border rounded-xl p-5">
          <p className="text-sm text-dark-muted">Instance Count</p>
          <p className="text-lg font-medium mt-1">{config.instance_count || "—"}</p>
        </div>
      </div>

      {/* Resource Topology */}
      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <h2 className="text-lg font-semibold mb-4">Resource Topology</h2>
        <div className="flex items-center gap-2 flex-wrap">
          <div className="bg-indigo-500/20 border border-indigo-500/30 rounded-lg px-3 py-2 text-sm">
            🌐 VPC
          </div>
          <span className="text-dark-muted">→</span>
          <div className="bg-blue-500/20 border border-blue-500/30 rounded-lg px-3 py-2 text-sm">
            🔀 Load Balancer
          </div>
          <span className="text-dark-muted">→</span>
          {Array.from({ length: Math.min(config.instance_count || 2, 5) }).map((_, i) => (
            <div key={i} className="bg-emerald-500/20 border border-emerald-500/30 rounded-lg px-3 py-2 text-sm">
              🖥️ Instance {i + 1}
            </div>
          ))}
        </div>
        <div className="flex items-center gap-2 mt-3">
          <div className="bg-purple-500/20 border border-purple-500/30 rounded-lg px-3 py-2 text-sm">
            📡 DNS ({env.slug}.infra.internal)
          </div>
          <div className="bg-yellow-500/20 border border-yellow-500/30 rounded-lg px-3 py-2 text-sm">
            📊 Monitoring
          </div>
          {driftCount > 0 && (
            <Link href="/drift" className="bg-red-500/20 border border-red-500/30 rounded-lg px-3 py-2 text-sm text-red-400">
              ⚠️ {driftCount} drift{driftCount > 1 ? "s" : ""}
            </Link>
          )}
        </div>
      </div>

      {/* Workflow History */}
      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <h2 className="text-lg font-semibold mb-4">Workflow History</h2>
        {workflows.length === 0 ? (
          <p className="text-dark-muted text-sm">No workflows for this environment.</p>
        ) : (
          <div className="space-y-2">
            {workflows.slice(0, 10).map((wf) => (
              <div key={wf.id} className="flex items-center justify-between py-2 border-b border-dark-border/50">
                <div>
                  <Link href={`/workflows/${wf.id}`} className="text-dark-accent hover:underline text-sm">
                    {wf.name}
                  </Link>
                  <span className="text-dark-muted text-xs ml-2">{wf.workflow_type}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-dark-muted">{new Date(wf.created_at).toLocaleString()}</span>
                  <StatusBadge status={wf.status} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
