"use client";
import { useEffect, useState } from "react";
import { fetchWorkflow, retryWorkflow, cancelWorkflow } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import { useParams } from "next/navigation";

export default function WorkflowDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const [workflow, setWorkflow] = useState<any>(null);
  const [actionLoading, setActionLoading] = useState("");

  useEffect(() => {
    const load = () => fetchWorkflow(id).then(setWorkflow).catch(console.error);
    load();
    const interval = setInterval(load, 2000);
    return () => clearInterval(interval);
  }, [id]);

  const handleRetry = async () => {
    setActionLoading("retry");
    try { await retryWorkflow(id); } catch (e) { console.error(e); }
    setActionLoading("");
  };

  const handleCancel = async () => {
    setActionLoading("cancel");
    try { await cancelWorkflow(id); } catch (e) { console.error(e); }
    setActionLoading("");
  };

  if (!workflow) return <div className="text-dark-muted">Loading...</div>;

  const steps = workflow.steps || [];
  const completedSteps = steps.filter((s: any) => s.status === "completed").length;
  const totalSteps = steps.length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{workflow.name}</h1>
          <p className="text-dark-muted text-sm">
            {workflow.workflow_type} · {workflow.initiated_by} · <StatusBadge status={workflow.status} />
          </p>
        </div>
        <div className="flex gap-2">
          {(workflow.status === "failed" || workflow.status === "cancelled") && (
            <button onClick={handleRetry} disabled={!!actionLoading}
              className="px-3 py-2 bg-dark-accent text-white rounded-lg text-sm hover:bg-indigo-600 disabled:opacity-50">
              {actionLoading === "retry" ? "..." : "Retry"}
            </button>
          )}
          {(workflow.status === "pending" || workflow.status === "running") && (
            <button onClick={handleCancel} disabled={!!actionLoading}
              className="px-3 py-2 bg-dark-border text-dark-text rounded-lg text-sm hover:bg-dark-border/80 disabled:opacity-50">
              {actionLoading === "cancel" ? "..." : "Cancel"}
            </button>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-dark-muted">Progress</span>
          <span className="text-sm font-medium">{completedSteps}/{totalSteps} steps</span>
        </div>
        <div className="w-full bg-dark-border rounded-full h-2">
          <div
            className={`h-2 rounded-full transition-all ${
              workflow.status === "failed" ? "bg-red-500" : "bg-dark-accent"
            }`}
            style={{ width: totalSteps > 0 ? `${(completedSteps / totalSteps) * 100}%` : "0%" }}
          />
        </div>
      </div>

      {/* Steps */}
      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <h2 className="text-lg font-semibold mb-4">Steps</h2>
        <div className="space-y-3">
          {steps.map((step: any) => (
            <div key={step.id} className="border border-dark-border/50 rounded-lg p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-xs bg-dark-border px-2 py-0.5 rounded font-mono">{step.step_order}</span>
                  <span className="font-medium text-sm">{step.name}</span>
                  <span className="text-xs text-dark-muted">{step.step_type}</span>
                </div>
                <StatusBadge status={step.status} />
              </div>

              {/* Logs / Output */}
              {step.output && Object.keys(step.output).length > 0 && (
                <details className="mt-3">
                  <summary className="text-xs text-dark-muted cursor-pointer hover:text-dark-text">
                    View output
                  </summary>
                  <pre className="mt-2 text-xs bg-dark-bg rounded-lg p-3 overflow-x-auto text-dark-muted">
                    {JSON.stringify(step.output, null, 2)}
                  </pre>
                </details>
              )}

              {step.error_message && (
                <p className="mt-2 text-xs text-red-400">Error: {step.error_message}</p>
              )}

              {step.started_at && (
                <p className="mt-1 text-xs text-dark-muted">
                  Started: {new Date(step.started_at).toLocaleString()}
                  {step.completed_at && ` · Completed: ${new Date(step.completed_at).toLocaleString()}`}
                </p>
              )}
            </div>
          ))}
          {steps.length === 0 && (
            <p className="text-dark-muted text-sm">No steps recorded yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}
