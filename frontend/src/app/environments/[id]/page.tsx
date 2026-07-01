"use client";
import { useEffect, useState, useCallback, useRef } from "react";
import { fetchEnvironment, fetchWorkflows, fetchWorkflow, fetchDriftRecords, decommissionEnvironment } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";
import { useParams } from "next/navigation";

export default function EnvironmentDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [env, setEnv] = useState<any>(null);
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [activeWorkflow, setActiveWorkflow] = useState<any>(null);
  const [driftCount, setDriftCount] = useState(0);
  const [decommissioning, setDecommissioning] = useState(false);
  const teamIdRef = useRef<string>("");

  const handleDecommission = async () => {
    if (!confirm("Are you sure you want to decommission this environment? This will destroy all resources.")) return;
    setDecommissioning(true);
    try {
      await decommissionEnvironment(id);
    } catch (e) { console.error(e); }
    setDecommissioning(false);
  };

  // Load environment once and extract team_id
  const loadEnv = useCallback(async () => {
    try {
      const e = await fetchEnvironment(id);
      setEnv(e);
      if (e?.team_id) teamIdRef.current = e.team_id;
    } catch (e) { console.error(e); }
  }, [id]);

  // Load workflows + active workflow detail
  const loadWorkflows = useCallback(async () => {
    const teamId = teamIdRef.current;
    if (!teamId) return;

    try {
      const wfs = (await fetchWorkflows(teamId)) || [];
      const envWorkflows = wfs.filter((w: any) => w.environment_id === id);
      setWorkflows(envWorkflows);

      // Find active workflow (running or pending)
      const active = envWorkflows.find((w: any) => w.status === "running" || w.status === "pending");
      if (active) {
        const detail = await fetchWorkflow(active.id);
        setActiveWorkflow(detail);
      } else {
        // Check if we had an active one that just completed — clear it
        setActiveWorkflow((prev: any) => {
          if (prev && envWorkflows.find((w: any) => w.id === prev.id)?.status !== "running" && envWorkflows.find((w: any) => w.id === prev.id)?.status !== "pending") {
            return null;
          }
          return prev;
        });
      }
    } catch (e) { console.error(e); }
  }, [id]);

  // Poll active workflow steps every 2 seconds for real-time updates
  const pollActiveSteps = useCallback(async () => {
    if (!activeWorkflow?.id) return;
    try {
      const detail = await fetchWorkflow(activeWorkflow.id);
      setActiveWorkflow(detail);
      // If workflow completed or failed, trigger a full workflow list reload
      if (detail.status !== "running" && detail.status !== "pending") {
        loadWorkflows();
      }
    } catch (e) { console.error(e); }
  }, [activeWorkflow?.id, loadWorkflows]);

  // Initial load
  useEffect(() => {
    loadEnv();
    fetchDriftRecords(id, true).then((r) => setDriftCount(r?.length || 0)).catch(() => {});
  }, [id, loadEnv]);

  // Once we have env/teamId, start workflow polling
  useEffect(() => {
    if (!teamIdRef.current && env?.team_id) {
      teamIdRef.current = env.team_id;
    }
    if (!teamIdRef.current) return;

    loadWorkflows();
    const interval = setInterval(loadWorkflows, 5000);
    return () => clearInterval(interval);
  }, [env?.team_id, loadWorkflows]);

  // Poll active workflow steps at 2s interval for real-time step progress
  useEffect(() => {
    if (!activeWorkflow?.id) return;
    const interval = setInterval(pollActiveSteps, 2000);
    return () => clearInterval(interval);
  }, [activeWorkflow?.id, pollActiveSteps]);

  if (!env) return <div className="text-dark-muted">Loading...</div>;

  const config = typeof env.config === "object" ? env.config : {};
  const steps = activeWorkflow?.steps || [];
  const completedSteps = steps.filter((s: any) => s.status === "completed").length;
  const failedSteps = steps.filter((s: any) => s.status === "failed").length;
  const totalSteps = steps.length;

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
          {env.status !== "decommissioned" && env.status !== "decommissioning" && (
            <button
              onClick={handleDecommission}
              disabled={decommissioning}
              className="px-4 py-2 bg-red-600/20 border border-red-600/40 text-red-400 rounded-lg text-sm hover:bg-red-600/30 disabled:opacity-50"
            >
              {decommissioning ? "..." : "Decommission"}
            </button>
          )}
          <StatusBadge status={env.status} />
        </div>
      </div>

      {/* Active Workflow Progress */}
      {activeWorkflow && (
        <div className={`bg-dark-card border rounded-xl p-5 ${
          activeWorkflow.status === "failed" ? "border-red-500/30" : "border-indigo-500/30"
        }`}>
          <div className="flex items-center justify-between mb-3">
            <div>
              <Link href={`/workflows/${activeWorkflow.id}`} className="font-medium text-dark-accent hover:underline">
                {activeWorkflow.name}
              </Link>
              <p className="text-xs text-dark-muted mt-0.5">{activeWorkflow.workflow_type} · {activeWorkflow.initiated_by}</p>
            </div>
            <StatusBadge status={activeWorkflow.status} />
          </div>

          {/* Progress bar */}
          <div className="mb-4">
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs text-dark-muted">Progress</span>
              <span className="text-xs font-medium">
                {completedSteps}/{totalSteps} steps
                {failedSteps > 0 && <span className="text-red-400 ml-1">({failedSteps} failed)</span>}
              </span>
            </div>
            <div className="w-full bg-dark-border rounded-full h-2">
              <div
                className={`h-2 rounded-full transition-all duration-500 ${
                  failedSteps > 0 ? "bg-red-500" : "bg-dark-accent"
                }`}
                style={{ width: totalSteps > 0 ? `${(completedSteps / totalSteps) * 100}%` : "0%" }}
              />
            </div>
          </div>

          {/* Step list */}
          <div className="space-y-1.5">
            {steps.map((step: any) => (
              <div key={step.id} className={`flex items-center justify-between py-2 px-3 rounded-lg ${
                step.status === "running" ? "bg-blue-500/10 border border-blue-500/20" :
                step.status === "failed" ? "bg-red-500/10 border border-red-500/20" :
                "bg-dark-bg/50"
              }`}>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-mono text-dark-muted w-5">{step.step_order}</span>
                  <span className="text-sm">{step.name.replace(/_/g, " ")}</span>
                  <span className="text-xs text-dark-muted">{step.step_type}</span>
                </div>
                <div className="flex items-center gap-2">
                  {step.status === "running" && (
                    <span className="inline-block w-2 h-2 bg-blue-400 rounded-full animate-pulse" />
                  )}
                  <StatusBadge status={step.status} />
                </div>
              </div>
            ))}
            {steps.length === 0 && activeWorkflow.status === "pending" && (
              <p className="text-xs text-dark-muted py-2">Waiting for workflow to start...</p>
            )}
          </div>

          {/* Error message */}
          {activeWorkflow.error_message && (
            <div className="mt-3 p-3 bg-red-500/10 border border-red-500/20 rounded-lg">
              <p className="text-xs text-red-400">{activeWorkflow.error_message}</p>
            </div>
          )}
        </div>
      )}

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
            VPC
          </div>
          <span className="text-dark-muted">→</span>
          <div className="bg-blue-500/20 border border-blue-500/30 rounded-lg px-3 py-2 text-sm">
            Load Balancer
          </div>
          <span className="text-dark-muted">→</span>
          {Array.from({ length: Math.min(config.instance_count || 2, 5) }).map((_, i) => (
            <div key={i} className="bg-emerald-500/20 border border-emerald-500/30 rounded-lg px-3 py-2 text-sm">
              Instance {i + 1}
            </div>
          ))}
        </div>
        <div className="flex items-center gap-2 mt-3">
          <div className="bg-purple-500/20 border border-purple-500/30 rounded-lg px-3 py-2 text-sm">
            DNS ({env.slug}.infra.internal)
          </div>
          <div className="bg-yellow-500/20 border border-yellow-500/30 rounded-lg px-3 py-2 text-sm">
            Monitoring
          </div>
          {driftCount > 0 && (
            <Link href="/drift" className="bg-red-500/20 border border-red-500/30 rounded-lg px-3 py-2 text-sm text-red-400">
              {driftCount} drift{driftCount > 1 ? "s" : ""}
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
            {workflows.map((wf) => (
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
