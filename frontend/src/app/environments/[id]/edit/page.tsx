"use client";
import { useEffect, useState } from "react";
import { fetchEnvironment, updateEnvironment } from "@/lib/api";
import { useParams, useRouter } from "next/navigation";

export default function EditEnvironmentPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;
  const [env, setEnv] = useState<any>(null);
  const [form, setForm] = useState({ instance_size: "", instance_count: 1 });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchEnvironment(id).then((e) => {
      setEnv(e);
      const config = typeof e.config === "object" ? e.config : {};
      setForm({
        instance_size: config.instance_size || "t3.medium",
        instance_count: config.instance_count || 2,
      });
    });
  }, [id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const config = { ...(typeof env.config === "object" ? env.config : {}), ...form };
      await updateEnvironment(id, { config: JSON.stringify(config) === "{}" ? config : config });
      router.push(`/environments/${id}`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  if (!env) return <div className="text-dark-muted">Loading...</div>;

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">Edit: {env.name}</h1>

      <form onSubmit={handleSubmit} className="bg-dark-card border border-dark-border rounded-xl p-6 space-y-5">
        {error && <div className="p-3 bg-red-500/20 border border-red-500/30 rounded-lg text-red-400 text-sm">{error}</div>}

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

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={loading}
            className="flex-1 bg-dark-accent text-white rounded-lg py-2 text-sm font-medium hover:bg-indigo-600 transition-colors disabled:opacity-50"
          >
            {loading ? "Saving..." : "Save Changes"}
          </button>
          <button
            type="button"
            onClick={() => router.back()}
            className="px-4 py-2 bg-dark-border text-dark-text rounded-lg text-sm hover:bg-dark-border/80"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
