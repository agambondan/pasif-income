"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import AccountSelector, {
    type ConnectedAccount,
} from "@/components/account-selector";

type GenerationJob = {
    id: string;
    niche: string;
    topic: string;
    title?: string;
    description?: string;
    pin_comment?: string;
    video_path?: string;
    status: "queued" | "running" | "completed" | "failed";
    current_stage: string;
    progress_pct: number;
    error?: string;
    scheduled_at?: string | null;
    created_at: string;
    updated_at: string;
};

type DistributionJob = {
    id: number;
    generation_job_id: string;
    account_id: string;
    platform: string;
    status: "pending" | "uploading" | "completed" | "failed";
    status_detail: string;
    external_id: string;
    error: string;
    scheduled_at?: string | null;
    created_at: string;
    updated_at: string;
};

type VoiceTypeOption = {
    id: string;
    label: string;
    language: string;
    tld: string;
};

const API_BASE_URL = "";

const jobStatusStyles: Record<GenerationJob["status"], string> = {
    queued: "bg-amber-500/15 text-amber-300 border-amber-500/30",
    running: "bg-blue-500/15 text-blue-300 border-blue-500/30",
    completed: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
    failed: "bg-red-500/15 text-red-300 border-red-500/30",
};

function formatTime(value?: string | null) {
    if (!value) return "-";
    const date = new Date(value);
    return isNaN(date.getTime()) ? value : date.toLocaleString();
}

export default function Dashboard() {
    const [jobs, setJobs] = useState<GenerationJob[]>([]);
    const [loading, setLoading] = useState(true);
    const [isGenerating, setIsGenerating] = useState(false);
    const [backendOnline, setBackendOnline] = useState(false);
    const [statusMessage, setStatusMessage] = useState<string | null>(null);
    const [lastUpdated, setLastUpdated] = useState("Never");
    const [niche, setNiche] = useState("stoicism");
    const [topic, setTopic] = useState("how to control your mind");
    const [voiceType, setVoiceType] = useState("en-US-Standard-A");
    const [voiceTypes, setVoiceTypes] = useState<VoiceTypeOption[]>([]);
    const [scheduleMode, setScheduleMode] = useState<"immediate" | "drip_feed" | "prime_time">("immediate");
    const [dripIntervalDays, setDripIntervalDays] = useState(1);
    const [accounts, setAccounts] = useState<ConnectedAccount[]>([]);
    const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
    const [selectedJobId, setSelectedJobId] = useState<string>("");
    const [distributionJobs, setDistributionJobs] = useState<DistributionJob[]>([]);
    
    const selectedJobIdRef = useRef("");

    const fetchAccounts = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/accounts`, { credentials: "include" });
            if (res.ok) setAccounts(await res.json() || []);
        } catch (err) { console.error("Failed to fetch accounts:", err); }
    }, []);

    const fetchVoiceTypes = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/voice-types`, { credentials: "include" });
            if (res.ok) setVoiceTypes(await res.json() || []);
        } catch (err) { console.error("Failed to fetch voice types:", err); }
    }, []);

    const fetchDistributionJobs = useCallback(async (jobId: string) => {
        if (!jobId) { setDistributionJobs([]); return; }
        try {
            const res = await fetch(`${API_BASE_URL}/api/jobs/${jobId}/distributions`, { credentials: "include" });
            if (res.ok) setDistributionJobs(await res.json() || []);
        } catch (err) { console.error("Failed to fetch distribution jobs:", err); }
    }, []);

    const fetchJobs = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/jobs`, { credentials: "include" });
            if (res.ok) {
                const nextJobs = await res.json() || [];
                setJobs(nextJobs);
                if (!selectedJobIdRef.current && nextJobs.length > 0) {
                    selectedJobIdRef.current = nextJobs[0].id;
                    setSelectedJobId(nextJobs[0].id);
                    void fetchDistributionJobs(nextJobs[0].id);
                }
            }
        } catch (err) { console.error("Failed to fetch jobs:", err); }
    }, [fetchDistributionJobs]);

    const fetchBackendHealth = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/health`, { credentials: "include" });
            setBackendOnline(res.ok);
        } catch (err) { setBackendOnline(false); }
    }, []);

    const refreshAll = useCallback(async () => {
        setLoading(true);
        await Promise.all([fetchBackendHealth(), fetchJobs(), fetchAccounts(), fetchVoiceTypes()]);
        if (selectedJobIdRef.current) await fetchDistributionJobs(selectedJobIdRef.current);
        setLoading(false);
        setLastUpdated(new Date().toLocaleString());
    }, [fetchBackendHealth, fetchJobs, fetchAccounts, fetchVoiceTypes, fetchDistributionJobs]);

    const pollState = useCallback(async () => {
        await Promise.all([fetchBackendHealth(), fetchJobs(), fetchAccounts()]);
        if (selectedJobIdRef.current) await fetchDistributionJobs(selectedJobIdRef.current);
        setLastUpdated(new Date().toLocaleString());
    }, [fetchBackendHealth, fetchJobs, fetchAccounts, fetchDistributionJobs]);

    useEffect(() => {
        refreshAll();
        const interval = window.setInterval(pollState, 10000);
        return () => window.clearInterval(interval);
    }, [refreshAll, pollState]);

    const handleSelectJob = (jobId: string) => {
        selectedJobIdRef.current = jobId;
        setSelectedJobId(jobId);
        void fetchDistributionJobs(jobId);
    };

    const cancelJob = async (id: string) => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/jobs/${id}/cancel`, { method: "POST", credentials: "include" });
            if (res.ok) {
                setStatusMessage("Job cancelled successfully");
                void fetchJobs();
            }
        } catch (err) { console.error("Failed to cancel job:", err); }
    };

    const retryJob = async (id: string) => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/jobs/${id}/retry`, { method: "POST", credentials: "include" });
            if (res.ok) {
                setStatusMessage("Retry initiated");
                void fetchJobs();
            }
        } catch (err) { console.error("Failed to retry job:", err); }
    };

    const startGeneration = async () => {
        try {
            if (!topic.trim()) { setStatusMessage("Topic is required"); return; }
            if (selectedAccounts.length === 0) { setStatusMessage("Select at least one destination account"); return; }
            setIsGenerating(true);
            setStatusMessage(null);

            const destinations = selectedAccounts.map((id) => {
                const acc = accounts.find((a) => a.id === id);
                return { platform: acc?.platform_id || "unknown", account_id: id };
            });

            const res = await fetch(`${API_BASE_URL}/api/generate`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ niche, topic, voice_type: voiceType, destinations, schedule_mode: scheduleMode, drip_interval_days: dripIntervalDays }),
            });
            if (res.ok) {
                const job = await res.json();
                setStatusMessage(`Job ${job.id} queued`);
                handleSelectJob(job.id);
                void fetchJobs();
            }
        } catch (err) { setStatusMessage("Failed to start generation"); }
        finally { setIsGenerating(false); }
    };

    const selectedJob = jobs.find(j => j.id === selectedJobId);

    if (loading) {
        return (
            <div className='flex items-center justify-center min-h-[60vh]'>
                <div className='animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 shadow-[0_0_20px_rgba(59,130,246,0.3)]'></div>
            </div>
        );
    }

    return (
        <div className='space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-10'>
            <div className='flex flex-col md:flex-row justify-between items-end gap-6 border-l-4 border-blue-500 pl-6'>
                <div>
                    <h2 className='text-4xl font-black text-white tracking-tighter uppercase'>Operations</h2>
                    <p className='text-zinc-500 mt-2 font-medium'>Control the AI production pipeline and monitor live jobs.</p>
                </div>
                <div className='flex gap-4'>
                    <button onClick={refreshAll} className='bg-zinc-900 hover:bg-zinc-800 px-6 py-3 rounded-2xl border border-white/5 text-xs font-bold transition-all active:scale-95 shadow-xl'>REFRESH DATA</button>
                    <div className={`px-6 py-3 rounded-2xl border text-xs font-black tracking-widest ${backendOnline ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-400" : "bg-red-500/10 border-red-500/20 text-red-400"}`}>
                        {backendOnline ? "● BACKEND ONLINE" : "● BACKEND OFFLINE"}
                    </div>
                </div>
            </div>

            <section className='grid gap-8 xl:grid-cols-[1.2fr_0.8fr]'>
                <div className='bg-card border border-white/5 rounded-[2.5rem] p-10 shadow-2xl relative overflow-hidden group h-fit'>
                    <div className='absolute -top-24 -right-24 w-64 h-64 bg-blue-500/5 rounded-full blur-3xl group-hover:bg-blue-500/10 transition-colors'></div>
                    <div className='relative z-10'>
                        <div className='flex items-center gap-3 mb-8'>
                            <div className='w-10 h-10 bg-blue-500/20 rounded-xl flex items-center justify-center text-xl shadow-lg border border-blue-500/30 text-blue-400'>⚡</div>
                            <h3 className='text-2xl font-bold text-white uppercase tracking-tight'>New Production Job</h3>
                        </div>

                        <div className='grid gap-6 md:grid-cols-2 mb-8'>
                            <div className='space-y-2'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>Niche Architecture</label>
                                <input value={niche} onChange={(e) => setNiche(e.target.value)} className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10' placeholder='stoicism' />
                            </div>
                            <div className='space-y-2'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>Content Concept</label>
                                <input value={topic} onChange={(e) => setTopic(e.target.value)} className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10' placeholder='how to control your mind' />
                            </div>
                        </div>

                        <div className='grid gap-6 md:grid-cols-2 mb-8'>
                            <div className='space-y-2'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>Voice Preset</label>
                                <select value={voiceType} onChange={(e) => setVoiceType(e.target.value)} className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'>
                                    {voiceTypes.map((v) => <option key={v.id} value={v.id}>{v.label} · {v.id}</option>)}
                                </select>
                            </div>
                            <div className='space-y-2'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>Scheduling Mode</label>
                                <select value={scheduleMode} onChange={(e) => setScheduleMode(e.target.value as any)} className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'>
                                    <option value='immediate'>Immediate</option>
                                    <option value='drip_feed'>Drip Feed</option>
                                    <option value='prime_time'>Prime Time</option>
                                </select>
                            </div>
                        </div>

                        <div className='mb-10'>
                            <AccountSelector 
                                title='Distribution Matrix' 
                                subtitle='Choose accounts to publish.' 
                                accounts={accounts} 
                                selectedIds={selectedAccounts} 
                                onChange={setSelectedAccounts} 
                                emptyMessage='No connected accounts found. Go to Integrations to connect.'
                            />
                        </div>

                        <button onClick={startGeneration} disabled={isGenerating} className='w-full rounded-2xl bg-gradient-to-r from-blue-600 to-blue-500 py-5 font-black text-white text-sm uppercase tracking-[0.2em] transition-all hover:from-blue-500 hover:to-blue-400 active:scale-[0.98] disabled:opacity-50 shadow-xl shadow-blue-900/20'>
                            {isGenerating ? "INITIALIZING..." : "EXECUTE PRODUCTION"}
                        </button>

                        {statusMessage && <div className='mt-6 rounded-2xl border border-blue-500/20 bg-blue-500/10 px-6 py-4 text-xs font-bold text-blue-400 text-center animate-bounce'>{statusMessage}</div>}
                    </div>
                </div>

                <div className='bg-card border border-white/5 rounded-[2.5rem] p-8 shadow-2xl flex flex-col'>
                    <div className='flex items-center gap-3 mb-8'>
                        <div className='w-10 h-10 bg-emerald-500/20 rounded-xl flex items-center justify-center text-xl shadow-lg border border-emerald-500/30 text-emerald-400'>📊</div>
                        <h3 className='text-2xl font-bold text-white uppercase tracking-tight'>Active Jobs</h3>
                    </div>

                    <div className='space-y-4 flex-1 overflow-auto max-h-[600px] pr-2'>
                        {jobs.map((job) => (
                            <div key={job.id} onClick={() => handleSelectJob(job.id)} className={`rounded-2xl border bg-black/40 p-5 transition-all cursor-pointer ${selectedJobId === job.id ? "border-emerald-500/40 shadow-lg shadow-emerald-500/5" : "border-white/5 hover:border-white/20"}`}>
                                <div className='flex items-start justify-between gap-4'>
                                    <div>
                                        <p className='font-black text-white text-sm uppercase tracking-tight'>{job.niche}</p>
                                        <p className='text-[10px] text-zinc-500 font-medium mt-1 uppercase tracking-widest'>{job.topic}</p>
                                    </div>
                                    <span className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest ${jobStatusStyles[job.status]}`}>{job.status}</span>
                                </div>
                                
                                {job.status === "running" && (
                                    <div className="mt-4 space-y-2">
                                        <div className="flex justify-between text-[8px] font-black text-blue-400 uppercase tracking-[0.2em]">
                                            <span>{job.current_stage || "Processing"}</span>
                                            <span>{job.progress_pct}%</span>
                                        </div>
                                        <div className="w-full bg-zinc-900 h-1 rounded-full overflow-hidden">
                                            <div className="bg-blue-500 h-full transition-all duration-500" style={{ width: `${job.progress_pct}%` }}></div>
                                        </div>
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>

                    {selectedJob && (
                        <div className='mt-8 pt-8 border-t border-white/5 space-y-6 animate-in fade-in duration-500'>
                            <div className="flex items-center justify-between">
                                <h4 className="text-lg font-black text-white uppercase tracking-tight">Job Control</h4>
                                <div className="flex gap-2">
                                    {(selectedJob.status === "queued" || selectedJob.status === "running") && (
                                        <button onClick={() => cancelJob(selectedJob.id)} className="bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 text-red-400 px-4 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest transition-all">Cancel</button>
                                    )}
                                    {(selectedJob.status === "failed") && (
                                        <button onClick={() => retryJob(selectedJob.id)} className="bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 text-blue-400 px-4 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest transition-all">Retry</button>
                                    )}
                                </div>
                            </div>
                            
                            <div className='space-y-3'>
                                {distributionJobs.map((dist) => (
                                    <div key={dist.id} className='rounded-2xl border border-white/5 bg-black/40 p-4'>
                                        <div className='flex items-center justify-between mb-2'>
                                            <p className='text-sm font-black text-white uppercase'>{dist.platform}</p>
                                            <span className={`text-[9px] font-black uppercase px-2 py-1 rounded-lg border ${dist.status === 'completed' ? 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30' : 'bg-amber-500/15 text-amber-300 border-amber-500/30'}`}>{dist.status}</span>
                                        </div>
                                        <p className='text-[10px] text-zinc-500 font-bold uppercase tracking-widest'>{dist.status_detail || "Queued"}</p>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </section>
        </div>
    );
}
