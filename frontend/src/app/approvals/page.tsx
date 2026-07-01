"use client";
import { useEffect, useState } from "react";
import { fetchApprovals, approveApproval, rejectApproval } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";

export default function ApprovalsPage() {
  const [approvals, setApprovals] = useState<any[]>([]);
  const [approverName, setApproverName] = useState("admin");
  const [reasons, setReasons] = useState<Record<string, string>>({});
  const [loadingId, setLoadingId] = useState("");

  useEffect(() => {
    const load = () => fetchApprovals().then(setApprovals).catch(console.error);
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleApprove = async (id: string) => {
    setLoadingId(id);
    try {
      await approveApproval(id, reasons[id] || `Approved by ${approverName}`);
      setApprovals((prev) => prev.filter((a) => a.id !== id));
    } catch (e) { console.error(e); }
    setLoadingId("");
  };

  const handleReject = async (id: string) => {
    setLoadingId(id);
    try {
      await rejectApproval(id, reasons[id] || `Rejected by ${approverName}`);
      setApprovals((prev) => prev.filter((a) => a.id !== id));
    } catch (e) { console.error(e); }
    setLoadingId("");
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Pending Approvals</h1>
        <div className="flex items-center gap-2">
          <label className="text-sm text-dark-muted">Approver:</label>
          <input
            type="text"
            value={approverName}
            onChange={(e) => setApproverName(e.target.value)}
            className="bg-dark-card border border-dark-border rounded-lg px-3 py-1 text-sm focus:outline-none focus:border-dark-accent w-40"
            placeholder="Your name"
          />
        </div>
      </div>

      {approvals.length === 0 ? (
        <div className="bg-dark-card border border-dark-border rounded-xl p-10 text-center">
          <p className="text-4xl mb-3">✅</p>
          <p className="text-dark-muted">No pending approvals. All clear!</p>
        </div>
      ) : (
        <div className="space-y-4">
          {approvals.map((approval) => (
            <div key={approval.id} className="bg-dark-card border border-dark-border rounded-xl p-5">
              <div className="flex items-start justify-between">
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <StatusBadge status={approval.status} />
                    <span className="text-sm text-dark-muted">Assigned to: {approval.assigned_to}</span>
                  </div>
                  <p className="text-sm">
                    Requested by <span className="font-medium">{approval.requested_by}</span>
                  </p>
                  <p className="text-xs text-dark-muted mt-1">
                    Workflow: {approval.workflow_id}
                  </p>
                  {approval.expires_at && (
                    <p className="text-xs text-yellow-400 mt-1">
                      Expires: {new Date(approval.expires_at).toLocaleString()}
                    </p>
                  )}
                </div>
              </div>

              <div className="mt-4 space-y-3">
                <input
                  type="text"
                  placeholder="Reason (optional)"
                  value={reasons[approval.id] || ""}
                  onChange={(e) => setReasons({ ...reasons, [approval.id]: e.target.value })}
                  className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
                />
                <div className="flex gap-3">
                  <button
                    onClick={() => handleApprove(approval.id)}
                    disabled={loadingId === approval.id}
                    className="px-4 py-2 bg-emerald-600 text-white rounded-lg text-sm hover:bg-emerald-700 disabled:opacity-50"
                  >
                    {loadingId === approval.id ? "..." : "✅ Approve"}
                  </button>
                  <button
                    onClick={() => handleReject(approval.id)}
                    disabled={loadingId === approval.id}
                    className="px-4 py-2 bg-red-600 text-white rounded-lg text-sm hover:bg-red-700 disabled:opacity-50"
                  >
                    {loadingId === approval.id ? "..." : "❌ Reject"}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
