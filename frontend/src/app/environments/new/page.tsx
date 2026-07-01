"use client";
import { useState, useEffect } from "react";
import { createEnvironment, fetchTeams } from "@/lib/api";
import { useRouter } from "next/navigation";

export default function NewEnvironmentPage() {
  const router = useRouter();
  const [teams, setTeams] = useState<any[]>([]);
  const [form, setForm] = useState({
    name: "",
    team_id: "",
    slug: "",
    provider: "aws",
    region: "us-east-1",
    tier: "development",
    instance_size: "t3.medium",
    instance_count: 2,
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchTeams().then((t) => {
      setTeams(t || []);
      if (t?.length) setForm((f) => ({ ...f, team_id: t[0].id }));
    });
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await createEnvironment(form.team_id, {
        name: form.name,
        slug: form.slug || form.name.toLowerCase().replace(/\s+/g, "-"),
        provider: form.provider,
        region: form.region,
        config: {
          tier: form.tier,
          instance_size: form.instance_size,
          instance_count: form.instance_count,
        },
      });
      router.push("/environments");
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">New Environment</h1>

      <form onSubmit={handleSubmit} className="bg-dark-card border border-dark-border rounded-xl p-6 space-y-5">
        {error && <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg text-red-400 text-sm">{error}</div>}

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm text-dark-muted mb-1">Name</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
              placeholder="Production US"
              required
            />
          </div>
          <div>
            <label className="block text-sm text-dark-muted mb-1">Team</label>
            <select
              value={form.team_id}
              onChange={(e) => setForm({ ...form, team_id: e.target.value })}
              className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
            >
              {teams.map((t) => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm text-dark-muted mb-1">Tier</label>
            <select
              value={form.tier}
              onChange={(e) => setForm({ ...form, tier: e.target.value })}
              className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
            >
              <option value="development">Development</option>
              <option value="staging">Staging</option>
              <option value="production">Production</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-dark-muted mb-1">Region</label>
            <select
              value={form.region}
              onChange={(e) => setForm({ ...form, region: e.target.value })}
              className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
            >
              <option value="us-east-1">US East (N. Virginia)</option>
              <option value="us-west-2">US West (Oregon)</option>
              <option value="eu-west-1">EU West (Ireland)</option>
              <option value="ap-south-1">Asia Pacific (Mumbai)</option>
              <option value="ap-southeast-1">Asia Pacific (Singapore)</option>
            </select>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm text-dark-muted mb-1">Instance Size</label>
            <select
              value={form.instance_size}
              onChange={(e) => setForm({ ...form, instance_size: e.target.value })}
              className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
            >
              <option value="t3.small">t3.small (2 vCPU, 2 GB)</option>
              <option value="t3.medium">t3.medium (2 vCPU, 4 GB)</option>
              <option value="t3.large">t3.large (2 vCPU, 8 GB)</option>
              <option value="m5.large">m5.large (2 vCPU, 8 GB)</option>
              <option value="m5.xlarge">m5.xlarge (4 vCPU, 16 GB)</option>
              <option value="c5.2xlarge">c5.2xlarge (8 vCPU, 16 GB)</option>
            </select>
          </div>
          <div>
            <label className="block text-sm text-dark-muted mb-1">Instance Count</label>
            <input
              type="number"
              min={1}
              max={20}
              value={form.instance_count}
              onChange={(e) => setForm({ ...form, instance_count: parseInt(e.target.value) || 1 })}
              className="w-full bg-dark-bg border border-dark-border rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-dark-accent"
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-dark-accent text-white rounded-lg py-2 text-sm font-medium hover:bg-indigo-600 transition-colors disabled:opacity-50"
        >
          {loading ? "Creating..." : "Create Environment"}
        </button>
      </form>
    </div>
  );
}
