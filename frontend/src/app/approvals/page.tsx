"use client";
import { useEffect, useState } from "react";
import { fetchApprovals, fetchApprovalHistory, approveApproval, rejectApproval, fetchWorkflow, fetchEnvironment, signalWorkflow } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import Link from "next/link";

interface EnrichedApproval {
  approval: any;
  workflow: any | null;
  environment: any | null;
}

export default function ApprovalsPage() {
  const [pendingApprovals, setPendingApprovals] = useState<EnrichedApproval[]>([]);
  const [history, setHistory] = useState<any[]>([]);
  const [approverName, setApproverName] = useState("admin");
  const [reasons, setReasons] = useState<Record<string, string>>({});
  const [loadingId, setLoadingId] = useState("");

  // Load pending approvals with full details
  useEffect(() => {
    const load = async () => {
      try {
        const approvals = (await fetchApprovals()) || [];
        const enriched: EnrichedApproval[] = await Promise.all(
          approvals.map(async (approval: any) => {
            let workflow = null;
            let environment = null;
            try {
              workflow = await fetchWorkflow(approval.workflow_id);
              if (workflow?.environment_id) {
                environment = await fetchEnvironment(workflow.environment_id);
              }
            } catch (e) { /* ignore */ }
            return { approval, workflow, environment };
          })
        );
        setPendingApprovals(enriched);
      } catch (e) { console.error(e); }
    };
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, []);

  // Load decision history
  useEffect(() => {
    const load = () => fetchApprovalHistory().then((data) => setHistory(data || [])).catch(console.error);
    load();
    const interval = setInterval(load, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleApprove = async (item: EnrichedApproval) => {
    const id = item.approval.id;
    setLoadingId(id);
    try {
      const reason = reasons[id] || `Approved by ${approverName}`;
      await approveApproval(id, reason);
      // Signal the workflow to continue
      if (item.workflow?.id) {
        await signalWorkflow(item.workflow.id, { approved: true, reason });
      }
      // Move to history instead of removing
      setPendingApprovals((prev) => prev.filter((a) => a.approval.id !== id));
      // Refresh history
      fetchApprovalHistory().then((data) => setHistory(data || [])).catch(console.error);
    } catch (e) { console.error(e); }
    setLoadingId("");
  };

  const handleReject = async (item: EnrichedApproval) => {
    const id = item.approval.id;
    setLoadingId(id);
    try {
      const reason = reasons[id] || `Rejected by ${approverName}`;
      await rejectApproval(id, reason);
      if (item.workflow?.id) {
        await signalWorkflow(item.workflow.id, { approved: false, reason });
      }
      setPendingApprovals((prev) => prev.filter((a) => a.approval.id !== id));
      fetchApprovalHistory().then((data) => setHistory(data || [])).catch(console.error);
    } catch (e) { console.error(e); }
    setLoadingId("");
  };

  const decidedHistory = history.filter((a) => a.status !== "pending");

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Approvals</h1>
          <p className="text-sm text-dark-muted mt-1">
            Production environments require approval before provisioning
          </p>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-sm text-dark-muted">Approver:</label>
          <input
            type="text"
            value={approverName}
            onChange={(e) => setApproverName(e.target.value)}
            className="bg-dark-card border border-dark-border rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-dark-accent w-44"
            placeholder="Your name"
          />
        </div>
      </div>

      {/* Pending Approvals */}
      <div>
        <h2 className="text-lg font-semibold mb-4">
          Pending
          {pendingApprovals.length > 0 && (
            <span className="ml-2 px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded-full text-xs">
              {pendingApprovals.length}
            </span>
          )}
        </h2>

        {pendingApprovals.length === 0 ? (
          <div className="bg-dark-card border border-dark-border rounded-xl p-8 text-center">
            <p className="text-dark-muted">No pending approvals</p>
          </div>
        ) : (
          <div className="space-y-4">
            {pendingApprovals.map((item) => (
              <div key={item.approval.id} className="bg-dark-card border border-yellow-500/30 rounded-xl p-5">
                {/* Environment Details */}
                {item.environment && (
                  <div className="mb-4 p-4 bg-dark-bg rounded-lg border border-dark-border">
                    <div className="flex items-center justify-between mb-2">
                      <Link href={`/environments/${item.environment.id}`} className="font-medium text-dark-accent hover:underline">
                        {item.environment.name}
                      </Link>
                      <StatusBadge status={item.environment.status} />
                    </div>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
                      <div>
                        <span className="text-dark-muted">Provider:</span>
                        <span className="ml-1 font-medium">{item.environment.provider?.toUpperCase()}</span>
                      </div>
                      <div>
                        <span className="text-dark-muted">Region:</span>
                        <span className="ml-1">{item.environment.region || "—"}</span>
                      </div>
                      <div>
                        <span className="text-dark-muted">Tier:</span>
                        <span className="ml-1 text-red-400 font-medium">Production</span>
                      </div>
                      <div>
                        <span className="text-dark-muted">Slug:</span>
                        <span className="ml-1">{item.environment.slug}</span>
                      </div>
                    </div>
                    {item.environment.config && (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm mt-2">
                        <div>
                          <span className="text-dark-muted">Size:</span>
                          <span className="ml-1">{item.environment.config.instance_size || "—"}</span>
                        </div>
                        <div>
                          <span className="text-dark-muted">Count:</span>
                          <span className="ml-1">{item.environment.config.instance_count || "—"}</span>
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* Workflow Info */}
                {item.workflow && (
                  <div className="mb-3 flex items-center gap-3 text-sm">
                    <span className="text-dark-muted">Workflow:</span>
                    <Link href={`/workflows/${item.workflow.id}`} className="text-dark-accent hover:underline">
                      {item.workflow.name}
                    </Link>
                    <StatusBadge status={item.workflow.status} />
                  </div>
                )}

                {/* Approval Metadata */}
                <div className="flex items-center gap-4 mb-4 text-sm text-dark-muted">
                  <span>Requested by <span className="text-dark-text">{item.approval.requested_by || "system"}</span></span>
                  <span>Assigned to <span className="text-dark-text">{item.approval.assigned_to}</span></span>
                  <span className="text-xs">{new Date(item.approval.created_at).toLocaleString()}</span>
                </div>

                {/* Action */}
                <div className="flex items-center gap-3">
                  <input
                    type="text"
                    placeholder="Reason (optional)"
                    value={reasons[item.approval.id] || ""}
                    onChange={(e) => setReasons({ ...reasons, [item.approval.id]: e.target.value })}
                    className="flex-1 bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
                  />
                  <button
                    onClick={() => handleApprove(item)}
                    disabled={loadingId === item.approval.id}
                    className="px-4 py-2 bg-emerald-600 text-white rounded-lg text-sm hover:bg-emerald-700 disabled:opacity-50"
                  >
                    {loadingId === item.approval.id ? "..." : "Approve"}
                  </button>
                  <button
                    onClick={() => handleReject(item)}
                    disabled={loadingId === item.approval.id}
                    className="px-4 py-2 bg-red-600 text-white rounded-lg text-sm hover:bg-red-700 disabled:opacity-50"
                  >
                    {loadingId === item.approval.id ? "..." : "Reject"}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Decision History */}
      <div>
        <h2 className="text-lg font-semibold mb-4">Decision History</h2>
        {decidedHistory.length === 0 ? (
          <p className="text-dark-muted text-sm">No decisions recorded yet.</p>
        ) : (
          <div className="bg-dark-card border border-dark-border rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-dark-muted border-b border-dark-border bg-dark-bg/50">
                  <th className="text-left py-3 px-4">Status</th>
                  <th className="text-left py-3 px-4">Requested By</th>
                  <th className="text-left py-3 px-4">Assigned To</th>
                  <th className="text-left py-3 px-4">Reason</th>
                  <th className="text-left py-3 px-4">Decided At</th>
                </tr>
              </thead>
              <tbody>
                {decidedHistory.map((a) => (
                  <tr key={a.id} className="border-b border-dark-border/50 hover:bg-dark-border/20">
                    <td className="py-3 px-4"><StatusBadge status={a.status} /></td>
                    <td className="py-3 px-4">{a.requested_by || "system"}</td>
                    <td className="py-3 px-4">{a.assigned_to}</td>
                    <td className="py-3 px-4 text-dark-muted max-w-xs truncate">{a.decision_reason || "—"}</td>
                    <td className="py-3 px-4 text-dark-muted">
                      {a.decided_at ? new Date(a.decided_at).toLocaleString() : "—"}
                    </td>
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
