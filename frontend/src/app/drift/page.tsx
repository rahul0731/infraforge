"use client";
import { useEffect, useState, useCallback } from "react";
import { fetchAllDriftRecords, fetchDriftRecords, resolveDrift, fetchEnvironments, fetchTeams } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";

export default function DriftPage() {
  const [records, setRecords] = useState<any[]>([]);
  const [environments, setEnvironments] = useState<any[]>([]);
  const [selectedEnv, setSelectedEnv] = useState("all");
  const [showUnresolvedOnly, setShowUnresolvedOnly] = useState(false);
  const [loadingId, setLoadingId] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const [resolutions, setResolutions] = useState<Record<string, string>>({});

  useEffect(() => {
    fetchTeams().then((teams) => {
      if (teams?.length) {
        fetchEnvironments(teams[0].id).then((envs) => {
          setEnvironments(envs || []);
        });
      }
    });
  }, []);

  const loadRecords = useCallback(async () => {
    try {
      let data: any[];
      if (selectedEnv === "all") {
        data = (await fetchAllDriftRecords(showUnresolvedOnly)) || [];
      } else {
        data = (await fetchDriftRecords(selectedEnv, showUnresolvedOnly)) || [];
      }
      setRecords(data);
    } catch (e) { console.error(e); }
  }, [selectedEnv, showUnresolvedOnly]);

  useEffect(() => {
    loadRecords();
    const interval = setInterval(loadRecords, 10000);
    return () => clearInterval(interval);
  }, [loadRecords]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadRecords();
    setRefreshing(false);
  };

  const handleResolve = async (id: string) => {
    const resolution = resolutions[id] || "manual_fix";
    setLoadingId(id);
    try {
      await resolveDrift(id, resolution);
      await loadRecords();
    } catch (e) { console.error(e); }
    setLoadingId("");
  };

  // Find environment name by ID
  const getEnvName = (envId: string) => {
    const env = environments.find((e) => e.id === envId);
    return env?.name || envId.slice(0, 8);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Infrastructure Drift</h1>
        <button
          onClick={handleRefresh}
          disabled={refreshing}
          className="px-4 py-2 bg-dark-card border border-dark-border text-dark-text rounded-lg text-sm hover:bg-dark-border/80 disabled:opacity-50"
        >
          {refreshing ? "Refreshing..." : "Refresh"}
        </button>
      </div>

      {/* Explainer */}
      <div className="bg-dark-card border border-dark-border rounded-xl p-5">
        <h2 className="text-lg font-semibold mb-2">What is Drift?</h2>
        <p className="text-sm text-dark-muted leading-relaxed">
          Infrastructure drift occurs when the actual state of your cloud resources diverges from the
          desired state defined in your Infrastructure-as-Code. This can happen due to manual console changes,
          emergency hotfixes, auto-scaling events, or third-party integrations modifying resources directly.
        </p>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <select
          value={selectedEnv}
          onChange={(e) => setSelectedEnv(e.target.value)}
          className="bg-dark-card border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
        >
          <option value="all">All Environments</option>
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
        <span className="text-sm text-dark-muted ml-auto">
          {records.length} record{records.length !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Records */}
      {records.length === 0 ? (
        <div className="bg-dark-card border border-dark-border rounded-xl p-8 text-center">
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
                  <div className="flex items-center gap-3 mt-1 text-xs text-dark-muted">
                    <span>Env: {getEnvName(record.environment_id)}</span>
                    <span>Detected: {new Date(record.drift_detected_at).toLocaleString()}</span>
                  </div>
                </div>
                {record.resolved_at && (
                  <div className="text-right">
                    <StatusBadge status="completed" />
                    <p className="text-xs text-dark-muted mt-1">{record.resolution}</p>
                  </div>
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
