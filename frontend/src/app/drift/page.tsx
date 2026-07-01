"use client";
import { useEffect, useState } from "react";
import { fetchDriftRecords, resolveDrift, fetchEnvironments, fetchTeams } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";

export default function DriftPage() {
  const [records, setRecords] = useState<any[]>([]);
  const [environments, setEnvironments] = useState<any[]>([]);
  const [selectedEnv, setSelectedEnv] = useState("");
  const [showUnresolvedOnly, setShowUnresolvedOnly] = useState(true);
  const [loadingId, setLoadingId] = useState("");
  const [resolutions, setResolutions] = useState<Record<string, string>>({});

  useEffect(() => {
    fetchTeams().then((teams) => {
      if (teams?.length) {
        fetchEnvironments(teams[0].id).then((envs) => {
          setEnvironments(envs || []);
          if (envs?.length) setSelectedEnv(envs[0].id);
        });
      }
    });
  }, []);

  useEffect(() => {
    if (!selectedEnv) return;
    const load = () => fetchDriftRecords(selectedEnv, showUnresolvedOnly).then(setRecords).catch(console.error);
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, [selectedEnv, showUnresolvedOnly]);

  const handleResolve = async (id: string) => {
    const resolution = resolutions[id] || "manual_fix";
    setLoadingId(id);
    try {
      await resolveDrift(id, resolution);
      setRecords((prev) => prev.filter((r) => r.id !== id));
    } catch (e) { console.error(e); }
    setLoadingId("");
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Infrastructure Drift</h1>

      {/* Explainer */}
      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <h2 className="text-lg font-semibold mb-2">What is Drift?</h2>
        <p className="text-sm text-dark-muted leading-relaxed">
          Infrastructure drift occurs when the actual state of your cloud resources diverges from the
          desired state defined in your Infrastructure-as-Code (Terraform, CloudFormation, etc.).
          This can happen due to manual changes in the console, emergency hotfixes, auto-scaling events,
          or third-party integrations modifying resources directly. Drift detection compares your expected
          configuration against what is actually running and flags discrepancies.
        </p>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <select
          value={selectedEnv}
          onChange={(e) => setSelectedEnv(e.target.value)}
          className="bg-dark-card border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
        >
          {environments.map((env) => (
            <option key={env.id} value={env.id}>{env.name}</option>
          ))}
        </select>
        <label className="flex items-center gap-2 text-sm text-dark-muted cursor-pointer">
          <input
            type="checkbox"
            checked={showUnresolvedOnly}
            onChange={(e) => setShowUnresolvedOnly(e.target.checked)}
            className="rounded border-dark-border"
          />
          Unresolved only
        </label>
      </div>

      {/* Records */}
      {records.length === 0 ? (
        <div className="bg-dark-card border border-dark-border rounded-xl p-10 text-center">
          <p className="text-4xl mb-3">🎉</p>
          <p className="text-dark-muted">No drift detected. Your infrastructure matches desired state.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {records.map((record) => (
            <div key={record.id} className="bg-dark-card border border-dark-border rounded-xl p-5">
              <div className="flex items-start justify-between mb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{record.resource_type}</span>
                    <span className="text-dark-muted text-sm">/ {record.resource_id}</span>
                    <StatusBadge status={record.severity} />
                  </div>
                  <p className="text-xs text-dark-muted mt-1">
                    Detected: {new Date(record.drift_detected_at).toLocaleString()}
                  </p>
                </div>
                {record.resolved_at && (
                  <span className="text-xs text-emerald-400">
                    Resolved: {record.resolution}
                  </span>
                )}
              </div>

              {/* Desired vs Actual */}
              <div className="grid grid-cols-2 gap-4 mt-3">
                <div>
                  <p className="text-xs text-dark-muted mb-1 font-medium">Expected (Desired)</p>
                  <pre className="text-xs bg-emerald-500/10 border border-emerald-500/20 rounded-lg p-3 overflow-x-auto">
                    {JSON.stringify(record.expected_state, null, 2)}
                  </pre>
                </div>
                <div>
                  <p className="text-xs text-dark-muted mb-1 font-medium">Actual (Current)</p>
                  <pre className="text-xs bg-red-500/10 border border-red-500/20 rounded-lg p-3 overflow-x-auto">
                    {JSON.stringify(record.actual_state, null, 2)}
                  </pre>
                </div>
              </div>

              {/* Resolve */}
              {!record.resolved_at && (
                <div className="mt-4 flex items-center gap-3">
                  <select
                    value={resolutions[record.id] || "manual_fix"}
                    onChange={(e) => setResolutions({ ...resolutions, [record.id]: e.target.value })}
                    className="bg-dark-bg border border-dark-border rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-dark-accent"
                  >
                    <option value="manual_fix">Manual Fix</option>
                    <option value="auto_remediated">Auto Remediated</option>
                    <option value="accepted">Accepted (ignore)</option>
                    <option value="ignored">Ignored</option>
                  </select>
                  <button
                    onClick={() => handleResolve(record.id)}
                    disabled={loadingId === record.id}
                    className="px-4 py-1.5 bg-dark-accent text-white rounded-lg text-sm hover:bg-indigo-600 disabled:opacity-50"
                  >
                    {loadingId === record.id ? "..." : "Resolve"}
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
