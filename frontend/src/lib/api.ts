const API_BASE = "/api/v1";

// Default team/actor headers (in a real app, these come from auth context)
const DEFAULT_HEADERS: Record<string, string> = {
  "Content-Type": "application/json",
  "X-Actor": "admin",
};

function teamHeaders(teamId: string) {
  return { ...DEFAULT_HEADERS, "X-Team-ID": teamId };
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, options);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  if (res.status === 204) return null as T;
  return res.json();
}

// Teams
export const fetchTeams = () => request<any[]>(`${API_BASE}/teams`);
export const createTeam = (data: any) =>
  request<any>(`${API_BASE}/teams`, { method: "POST", headers: DEFAULT_HEADERS, body: JSON.stringify(data) });

// Environments
export const fetchEnvironments = (teamId: string) =>
  request<any[]>(`${API_BASE}/environments`, { headers: teamHeaders(teamId) });
export const fetchEnvironment = (id: string) =>
  request<any>(`${API_BASE}/environments/${id}`, { headers: DEFAULT_HEADERS });
export const createEnvironment = (teamId: string, data: any) =>
  request<any>(`${API_BASE}/environments`, { method: "POST", headers: teamHeaders(teamId), body: JSON.stringify(data) });
export const updateEnvironment = (id: string, data: any) =>
  request<any>(`${API_BASE}/environments/${id}`, { method: "PUT", headers: DEFAULT_HEADERS, body: JSON.stringify(data) });
export const deleteEnvironment = (id: string) =>
  request<any>(`${API_BASE}/environments/${id}`, { method: "DELETE", headers: DEFAULT_HEADERS });

// Workflows
export const fetchWorkflows = (teamId: string, status?: string) => {
  const params = status ? `?status=${status}` : "";
  return request<any[]>(`${API_BASE}/workflows${params}`, { headers: teamHeaders(teamId) });
};
export const fetchWorkflow = (id: string) =>
  request<any>(`${API_BASE}/workflows/${id}`, { headers: DEFAULT_HEADERS });
export const createWorkflow = (teamId: string, data: any) =>
  request<any>(`${API_BASE}/workflows`, { method: "POST", headers: teamHeaders(teamId), body: JSON.stringify(data) });
export const retryWorkflow = (id: string) =>
  request<any>(`${API_BASE}/workflows/${id}/retry`, { method: "POST", headers: DEFAULT_HEADERS });
export const cancelWorkflow = (id: string) =>
  request<any>(`${API_BASE}/workflows/${id}/cancel`, { method: "POST", headers: DEFAULT_HEADERS });
export const signalWorkflow = (id: string, data: { approved: boolean; reason: string }) =>
  request<any>(`${API_BASE}/workflows/${id}/signal`, { method: "POST", headers: DEFAULT_HEADERS, body: JSON.stringify(data) });

// Approvals
export const fetchApprovals = (assignedTo?: string) => {
  const params = assignedTo ? `?assigned_to=${assignedTo}` : "";
  return request<any[]>(`${API_BASE}/approvals${params}`);
};
export const approveApproval = (id: string, reason?: string) =>
  request<any>(`${API_BASE}/approvals/${id}/approve`, { method: "POST", headers: DEFAULT_HEADERS, body: JSON.stringify({ reason }) });
export const rejectApproval = (id: string, reason?: string) =>
  request<any>(`${API_BASE}/approvals/${id}/reject`, { method: "POST", headers: DEFAULT_HEADERS, body: JSON.stringify({ reason }) });

// Drift
export const fetchDriftRecords = (environmentId: string, unresolved?: boolean) => {
  const params = new URLSearchParams({ environment_id: environmentId });
  if (unresolved) params.set("unresolved", "true");
  return request<any[]>(`${API_BASE}/drift?${params}`);
};
export const reportDrift = (data: any) =>
  request<any>(`${API_BASE}/drift`, { method: "POST", headers: DEFAULT_HEADERS, body: JSON.stringify(data) });
export const resolveDrift = (id: string, resolution: string) =>
  request<any>(`${API_BASE}/drift/${id}/resolve`, { method: "POST", headers: DEFAULT_HEADERS, body: JSON.stringify({ resolution }) });

// Dashboard
export const fetchDashboardStats = (teamId?: string) => {
  const params = teamId ? `?team_id=${teamId}` : "";
  return request<any>(`${API_BASE}/dashboard/stats${params}`);
};

// Audit
export const fetchAuditLogs = (teamId?: string, limit?: number) => {
  const params = new URLSearchParams();
  if (teamId) params.set("team_id", teamId);
  if (limit) params.set("limit", String(limit));
  return request<any[]>(`${API_BASE}/audit?${params}`);
};
