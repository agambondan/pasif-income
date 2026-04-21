"use client";

import { Clip } from "@/types/clip";
import { useCallback, useEffect, useRef, useState } from "react";

type ConnectedAccount = {
    id: string;
    platform_id: string;
    display_name: string;
};

type GenerationJob = {
    id: string;
    niche: string;
    topic: string;
    title?: string;
    description?: string;
    pin_comment?: string;
    video_path?: string;
    status: "queued" | "running" | "completed" | "failed";
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

type TrendSignal = {
    query: string;
    source: string;
    score: number;
    link?: string;
    context?: string;
};

type IdeaSuggestion = {
    title: string;
    hook: string;
    angle: string;
    search_query: string;
    trend_source: string;
    score: number;
    reason: string;
};

type VoiceTypeOption = {
    id: string;
    label: string;
    language: string;
    tld: string;
};

type TrendResearchResult = {
    niche: string;
    seed: string;
    signals: TrendSignal[];
    ideas: IdeaSuggestion[];
    warnings?: string[];
    collected_at: string;
};

const API_BASE_URL = "";

const jobStatusStyles: Record<GenerationJob["status"], string> = {
    queued: "bg-amber-500/15 text-amber-300 border-amber-500/30",
    running: "bg-blue-500/15 text-blue-300 border-blue-500/30",
    completed: "bg-emerald-500/15 text-emerald-300 border-emerald-500/30",
    failed: "bg-red-500/15 text-red-300 border-red-500/30",
};

function formatTime(value?: string | null) {
    if (!value) {
        return "-";
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return value;
    }
    return date.toLocaleString();
}

export default function Dashboard() {
    const [clips, setClips] = useState<Clip[]>([]);
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
    const [scheduleMode, setScheduleMode] = useState<
        "immediate" | "drip_feed" | "prime_time"
    >("immediate");
    const [dripIntervalDays, setDripIntervalDays] = useState(1);
    const [accounts, setAccounts] = useState<ConnectedAccount[]>([]);
    const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
    const [selectedJobId, setSelectedJobId] = useState<string>("");
    const [distributionJobs, setDistributionJobs] = useState<DistributionJob[]>(
        [],
    );
    const [ideaSuggestions, setIdeaSuggestions] = useState<IdeaSuggestion[]>(
        [],
    );
    const [ideaWarnings, setIdeaWarnings] = useState<string[]>([]);
    const [isResearching, setIsResearching] = useState(false);
    const selectedJobIdRef = useRef("");

    const fetchAccounts = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/accounts`, {
                credentials: "include",
            });
            if (res.ok) {
                const data = await res.json();
                setAccounts(data || []);
            }
        } catch (err) {
            console.error("Failed to fetch accounts:", err);
        }
    }, []);

    const fetchClips = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/clips`, {
                credentials: "include",
            });
            if (!res.ok) {
                console.error(`Failed to fetch clips: ${res.status}`);
                return;
            }
            const data = await res.json();
            setClips(data || []);
        } catch (err) {
            console.error("Failed to fetch clips:", err);
        }
    }, []);

    const fetchVoiceTypes = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/voice-types`, {
                credentials: "include",
            });
            if (!res.ok) {
                return;
            }
            const data = (await res.json()) as VoiceTypeOption[];
            const options = data || [];
            setVoiceTypes(options);
            if (options.length === 0) return;
            setVoiceType((current) =>
                options.some((item) => item.id === current)
                    ? current
                    : options[0].id,
            );
        } catch (err) {
            console.error("Failed to fetch voice types:", err);
        }
    }, []);

    const fetchDistributionJobs = useCallback(async (jobId: string) => {
        if (!jobId) {
            setDistributionJobs([]);
            return;
        }
        try {
            const res = await fetch(
                `${API_BASE_URL}/api/jobs/${jobId}/distributions`,
                { credentials: "include" },
            );
            if (!res.ok) {
                return;
            }
            const data = await res.json();
            setDistributionJobs(data || []);
        } catch (err) {
            console.error("Failed to fetch distribution jobs:", err);
        }
    }, []);

    const fetchJobs = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/jobs`, {
                credentials: "include",
            });
            if (!res.ok) {
                console.error(`Failed to fetch jobs: ${res.status}`);
                return;
            }
            const data = await res.json();
            const nextJobs = data || [];
            setJobs(nextJobs);
            if (!selectedJobIdRef.current && nextJobs.length > 0) {
                selectedJobIdRef.current = nextJobs[0].id;
                setSelectedJobId(nextJobs[0].id);
                void fetchDistributionJobs(nextJobs[0].id);
            }
        } catch (err) {
            console.error("Failed to fetch jobs:", err);
        }
    }, [fetchDistributionJobs]);

    const fetchBackendHealth = useCallback(async () => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/health`, {
                credentials: "include",
            });
            setBackendOnline(res.ok);
        } catch (err) {
            console.error("Failed to fetch backend health:", err);
            setBackendOnline(false);
        }
    }, []);

    const refreshAll = useCallback(async () => {
        setLoading(true);
        await Promise.all([
            fetchBackendHealth(),
            fetchClips(),
            fetchJobs(),
            fetchAccounts(),
            fetchVoiceTypes(),
        ]);
        if (selectedJobIdRef.current) {
            await fetchDistributionJobs(selectedJobIdRef.current);
        }
        setLoading(false);
        setLastUpdated(new Date().toLocaleString());
    }, [
        fetchBackendHealth,
        fetchClips,
        fetchJobs,
        fetchAccounts,
        fetchVoiceTypes,
        fetchDistributionJobs,
    ]);

    const pollState = useCallback(async () => {
        await Promise.all([
            fetchBackendHealth(),
            fetchClips(),
            fetchJobs(),
            fetchAccounts(),
        ]);
        if (selectedJobIdRef.current) {
            await fetchDistributionJobs(selectedJobIdRef.current);
        }
        setLastUpdated(new Date().toLocaleString());
    }, [
        fetchBackendHealth,
        fetchClips,
        fetchJobs,
        fetchAccounts,
        fetchDistributionJobs,
    ]);

    useEffect(() => {
        Promise.resolve().then(() => refreshAll());
        const interval = window.setInterval(() => {
            void pollState();
        }, 10000);

        return () => window.clearInterval(interval);
    }, [refreshAll, pollState]);

    const handleSelectJob = (jobId: string) => {
        selectedJobIdRef.current = jobId;
        setSelectedJobId(jobId);
        void fetchDistributionJobs(jobId);
    };

    const updateStatus = async (id: string, status: Clip["status"]) => {
        try {
            await fetch(`${API_BASE_URL}/api/clips`, {
                method: "PATCH",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ id, status }),
            });
            await fetchClips();
        } catch (err) {
            console.error("Failed to update status:", err);
        }
    };

    const researchIdeas = async () => {
        try {
            setIsResearching(true);
            setStatusMessage(null);
            const res = await fetch(`${API_BASE_URL}/api/research/ideas`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ niche, limit: 5 }),
            });
            if (!res.ok) {
                throw new Error(await res.text());
            }
            const result = (await res.json()) as TrendResearchResult;
            setIdeaSuggestions(result.ideas || []);
            setIdeaWarnings(result.warnings || []);
            if (result.ideas?.[0]?.search_query) {
                setTopic(result.ideas[0].search_query);
            }
            setStatusMessage(
                `Found ${result.ideas?.length || 0} topic ideas for ${result.niche}`,
            );
        } catch (err) {
            console.error("Failed to research ideas:", err);
            setStatusMessage(
                err instanceof Error ? err.message : "Failed to research ideas",
            );
        } finally {
            setIsResearching(false);
        }
    };

    const startGeneration = async () => {
        try {
            if (!topic.trim()) {
                setStatusMessage("Topic is required");
                return;
            }
            setIsGenerating(true);
            setStatusMessage(null);

            const destinations = selectedAccounts.map((id) => {
                const acc = accounts.find((a) => a.id === id);
                return {
                    platform: acc?.platform_id || "unknown",
                    account_id: id,
                };
            });

            const res = await fetch(`${API_BASE_URL}/api/generate`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    niche,
                    topic,
                    voice_type: voiceType,
                    destinations,
                    schedule_mode: scheduleMode,
                    drip_interval_days: dripIntervalDays,
                }),
            });
            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || `HTTP ${res.status}`);
            }
            const job = await res.json();
            setStatusMessage(
                `Job ${job.id} queued for ${job.niche}: ${job.topic}`,
            );
            handleSelectJob(job.id);
            await Promise.all([fetchClips(), fetchJobs()]);
        } catch (err) {
            console.error("Failed to start generation:", err);
            setStatusMessage(
                err instanceof Error
                    ? err.message
                    : "Failed to start generation",
            );
        } finally {
            setIsGenerating(false);
        }
    };

    return (
        <div className='space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-10'>
            <div className='flex flex-col md:flex-row justify-between items-end gap-6 border-l-4 border-blue-500 pl-6'>
                <div>
                    <h2 className='text-4xl font-black text-white tracking-tighter uppercase'>
                        Operations
                    </h2>
                    <p className='text-zinc-500 mt-2 font-medium'>
                        Control the AI production pipeline, choose a voice preset, and monitor live jobs.
                    </p>
                </div>
                <div className='flex gap-4'>
                    <button
                        onClick={refreshAll}
                        className='bg-zinc-900 hover:bg-zinc-800 px-6 py-3 rounded-2xl border border-white/5 text-xs font-bold transition-all active:scale-95 shadow-xl'
                    >
                        REFRESH DATA
                    </button>
                    <div
                        className={`px-6 py-3 rounded-2xl border text-xs font-black tracking-widest ${backendOnline ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-400" : "bg-red-500/10 border-red-500/20 text-red-400"}`}
                    >
                        {backendOnline ? "● BACKEND ONLINE" : "● BACKEND OFFLINE"}
                    </div>
                    <div className='px-6 py-3 rounded-2xl border border-white/5 bg-zinc-900 text-[10px] font-bold tracking-widest text-zinc-500'>
                        UPDATED {lastUpdated}
                    </div>
                </div>
            </div>

            <section className='grid gap-8 xl:grid-cols-[1.2fr_0.8fr]'>
                <div className='bg-card border border-white/5 rounded-[2.5rem] p-10 shadow-2xl relative overflow-hidden group h-fit'>

                    <div className='absolute -top-24 -right-24 w-64 h-64 bg-blue-500/5 rounded-full blur-3xl group-hover:bg-blue-500/10 transition-colors'></div>

                    <div className='relative z-10'>
                        <div className='flex items-center gap-3 mb-8'>
                            <div className='w-10 h-10 bg-blue-500/20 rounded-xl flex items-center justify-center text-xl shadow-lg border border-blue-500/30 text-blue-400'>
                                ⚡
                            </div>
                            <h3 className='text-2xl font-bold text-white uppercase tracking-tight'>
                                New Production Job
                            </h3>
                        </div>

                        <div className='grid gap-6 md:grid-cols-2 mb-8'>
                            <div className='space-y-2'>
                                <label htmlFor="niche-input" className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>
                                    Niche Architecture
                                </label>
                                <input
                                    id="niche-input"
                                    value={niche}
                                    onChange={(e) => setNiche(e.target.value)}
                                    className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'
                                    placeholder='stoicism'
                                />
                            </div>
                            <div className='space-y-2'>
                                <label htmlFor="topic-input" className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>
                                    Content Concept
                                </label>
                                <input
                                    id="topic-input"
                                    value={topic}
                                    onChange={(e) => setTopic(e.target.value)}
                                    className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'
                                    placeholder='how to control your mind'
                                />
                            </div>
                        </div>

                        <div className='grid gap-6 md:grid-cols-2 mb-8'>
                            <div className='space-y-2'>
                                <label htmlFor="voice-type-input" className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>
                                    Voice Preset
                                </label>
                                <select
                                    id="voice-type-input"
                                    value={voiceType}
                                    onChange={(e) => setVoiceType(e.target.value)}
                                    className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'
                                >
                                    {voiceTypes.length > 0 ? (
                                        voiceTypes.map((option) => (
                                            <option key={option.id} value={option.id}>
                                                {option.label} · {option.id}
                                            </option>
                                        ))
                                    ) : (
                                        <>
                                            <option value='en-US-Standard-A'>English (US) · en-US-Standard-A</option>
                                            <option value='en-GB-Standard-A'>English (UK) · en-GB-Standard-A</option>
                                        </>
                                    )}
                                </select>
                                <div className='space-y-2 rounded-2xl border border-white/5 bg-black/20 px-4 py-3'>
                                    <p className='text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        Default follows `VOICE_TYPE` env or `en-US-Standard-A`.
                                    </p>
                                    <p className='text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        CLI: `go run cmd/creator/main.go --list-voice-types`
                                    </p>
                                    <p className='text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        Selected preset is forwarded to the voice adapter during generation.
                                    </p>
                                </div>
                            </div>
                            <div className='space-y-2'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>
                                    Scheduling Mode
                                </label>
                                <select
                                    value={scheduleMode}
                                    onChange={(e) =>
                                        setScheduleMode(
                                            e.target
                                                .value as typeof scheduleMode,
                                        )
                                    }
                                    className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'
                                >
                                    <option value='immediate'>Immediate</option>
                                    <option value='drip_feed'>Drip Feed</option>
                                    <option value='prime_time'>
                                        Prime Time
                                    </option>
                                </select>
                            </div>
                        </div>

                        <div className='grid gap-6 md:grid-cols-2 mb-8'>
                            <div className='space-y-2'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1'>
                                    Drip Interval Days
                                </label>
                                <input
                                    type='number'
                                    min={1}
                                    max={30}
                                    value={dripIntervalDays}
                                    onChange={(e) =>
                                        setDripIntervalDays(
                                            Number(e.target.value) || 1,
                                        )
                                    }
                                    className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10'
                                    placeholder='1'
                                />
                            </div>
                        </div>

                        <div className='mb-8 rounded-3xl border border-white/5 bg-black/30 p-5'>
                            <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4'>
                                <div>
                                    <p className='text-[10px] font-black text-zinc-500 uppercase tracking-widest'>
                                        Trend Research
                                    </p>
                                    <h4 className='text-lg font-black text-white uppercase tracking-tight'>
                                        Discover topic ideas from live signals
                                    </h4>
                                    <p className='mt-2 text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        Pick a suggestion to fill the content concept field.
                                    </p>
                                </div>
                                <button
                                    onClick={researchIdeas}
                                    disabled={isResearching}
                                    className='rounded-2xl bg-fuchsia-500/10 border border-fuchsia-500/20 px-5 py-3 text-[10px] font-black uppercase tracking-widest text-fuchsia-300 transition-all hover:bg-fuchsia-500/20 active:scale-95 disabled:opacity-50'
                                >
                                    {isResearching
                                        ? "RESEARCHING..."
                                        : "DISCOVER IDEAS"}
                                </button>
                            </div>

                            {ideaWarnings.length > 0 && (
                                <div className='mt-4 flex flex-wrap gap-2'>
                                    {ideaWarnings.slice(0, 3).map((warning) => (
                                        <span
                                            key={warning}
                                            className='rounded-full border border-amber-500/20 bg-amber-500/10 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-amber-300'
                                        >
                                            {warning}
                                        </span>
                                    ))}
                                </div>
                            )}

                            {ideaSuggestions.length > 0 ? (
                                <div className='mt-5 grid gap-3'>
                                    {ideaSuggestions.map((idea) => (
                                        <button
                                            key={`${idea.title}-${idea.search_query}`}
                                            onClick={() =>
                                                setTopic(idea.search_query)
                                            }
                                            className='text-left rounded-2xl border border-white/5 bg-black/40 p-4 transition-all hover:border-fuchsia-500/30 hover:bg-black/60 active:scale-[0.99]'
                                        >
                                            <div className='flex items-start justify-between gap-3'>
                                                <div>
                                                    <p className='text-sm font-black text-white uppercase tracking-tight'>
                                                        {idea.title}
                                                    </p>
                                                    <p className='mt-1 text-[10px] font-mono text-zinc-500 uppercase tracking-widest'>
                                                        {idea.trend_source} ·
                                                        score {idea.score}
                                                    </p>
                                                </div>
                                                <span className='rounded-lg border border-fuchsia-500/20 bg-fuchsia-500/10 px-2 py-1 text-[9px] font-black uppercase tracking-widest text-fuchsia-300'>
                                                    Use
                                                </span>
                                            </div>
                                            <p className='mt-3 text-xs text-zinc-300'>
                                                {idea.hook}
                                            </p>
                                            <p className='mt-2 text-[11px] text-zinc-500'>
                                                {idea.angle}
                                            </p>
                                        </button>
                                    ))}
                                </div>
                            ) : (
                                <div className='mt-5 rounded-2xl border border-dashed border-white/5 py-10 text-center'>
                                    <p className='text-zinc-600 text-sm font-bold uppercase tracking-widest'>
                                        No ideas yet
                                    </p>
                                </div>
                            )}
                        </div>

                        {accounts.length > 0 && (
                            <div className='mb-10'>
                                <span className='text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1 block mb-4 text-center'>
                                    Distribution Matrix
                                </span>
                                <p className='mb-4 text-center text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                    Select one or more connected accounts to publish the generated job.
                                </p>
                                <div className='flex flex-wrap justify-center gap-3'>
                                    {accounts.map((acc) => (
                                        <label
                                            key={acc.id}
                                            className={`flex items-center gap-3 border px-5 py-3 rounded-2xl cursor-pointer transition-all duration-300 ${selectedAccounts.includes(acc.id) ? "bg-emerald-500/10 border-emerald-500/50 shadow-lg shadow-emerald-500/5" : "bg-black/40 border-white/5 hover:border-white/20 hover:bg-black/60"}`}
                                        >
                                            <input
                                                type='checkbox'
                                                className='hidden'
                                                checked={selectedAccounts.includes(
                                                    acc.id,
                                                )}
                                                onChange={(e) => {
                                                    if (e.target.checked)
                                                        setSelectedAccounts([
                                                            ...selectedAccounts,
                                                            acc.id,
                                                        ]);
                                                    else
                                                        setSelectedAccounts(
                                                            selectedAccounts.filter(
                                                                (id) =>
                                                                    id !==
                                                                    acc.id,
                                                            ),
                                                        );
                                                }}
                                            />
                                            <div
                                                className={`w-5 h-5 rounded-lg border-2 flex items-center justify-center transition-all ${selectedAccounts.includes(acc.id) ? "bg-emerald-500 border-emerald-500 rotate-0" : "border-zinc-700 rotate-45"}`}
                                            >
                                                {selectedAccounts.includes(
                                                    acc.id,
                                                ) && (
                                                    <span className='text-black text-xs font-black'>
                                                        ✓
                                                    </span>
                                                )}
                                            </div>
                                            <div className='flex flex-col'>
                                                <span className='text-sm font-bold text-white'>
                                                    {acc.display_name}
                                                </span>
                                                <span className='text-[9px] text-zinc-500 font-black uppercase tracking-widest'>
                                                    {acc.platform_id}
                                                </span>
                                            </div>
                                        </label>
                                    ))}
                                </div>
                            </div>
                        )}

                        <button
                            onClick={startGeneration}
                            disabled={isGenerating}
                            className='w-full rounded-2xl bg-gradient-to-r from-blue-600 to-blue-500 py-5 font-black text-white text-sm uppercase tracking-[0.2em] transition-all hover:from-blue-500 hover:to-blue-400 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed shadow-xl shadow-blue-900/20 flex items-center justify-center gap-3'
                        >
                            {isGenerating ? (
                                <>
                                    <div className='animate-spin rounded-full h-4 w-4 border-2 border-white/30 border-t-white'></div>
                                    INITIALIZING...
                                </>
                            ) : (
                                "EXECUTE PRODUCTION"
                            )}
                        </button>

                        {statusMessage && (
                            <div 
                                data-testid="status-message"
                                className='mt-6 rounded-2xl border border-blue-500/20 bg-blue-500/10 px-6 py-4 text-xs font-bold text-blue-400 text-center animate-bounce'
                            >
                                {statusMessage}
                            </div>
                        )}
                    </div>
                </div>

                <div className='bg-card border border-white/5 rounded-[2.5rem] p-8 shadow-2xl flex flex-col'>
                    <div className='flex items-center gap-3 mb-8'>
                        <div className='w-10 h-10 bg-emerald-500/20 rounded-xl flex items-center justify-center text-xl shadow-lg border border-emerald-500/30 text-emerald-400'>
                            📊
                        </div>
                        <h3 className='text-2xl font-bold text-white uppercase tracking-tight'>
                            Active Jobs
                        </h3>
                    </div>

                    <div className='space-y-4 flex-1 overflow-auto max-h-[500px] pr-2'>
                        {jobs && jobs.length > 0 ? (
                            jobs.map((job) => (
                                <div
                                    key={job.id}
                                    onClick={() => handleSelectJob(job.id)}
                                    className={`rounded-2xl border bg-black/40 p-5 transition-all group/job cursor-pointer ${selectedJobId === job.id ? "border-emerald-500/40 shadow-lg shadow-emerald-500/5" : "border-white/5 hover:border-white/20"}`}
                                >
                                    <div className='flex items-start justify-between gap-4 mb-3'>
                                        <div>
                                            <p className='font-black text-white text-sm group-hover:text-blue-400 transition-colors uppercase tracking-tight'>
                                                {job.niche}
                                            </p>
                                            <p className='text-xs text-zinc-500 font-medium line-clamp-1 mt-1 uppercase tracking-widest'>
                                                {job.topic}
                                            </p>
                                            {job.scheduled_at && (
                                                <p className='text-[10px] text-emerald-400 font-bold uppercase tracking-widest mt-2'>
                                                    Scheduled{" "}
                                                    {formatTime(
                                                        job.scheduled_at,
                                                    )}
                                                </p>
                                            )}
                                        </div>
                                        <span
                                            className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest shadow-sm ${jobStatusStyles[job.status]}`}
                                        >
                                            {job.status}
                                        </span>
                                    </div>
                                    <div className='flex flex-col gap-1'>
                                        <p className='text-[10px] text-zinc-600 font-bold uppercase tracking-tighter'>
                                            {formatTime(job.created_at)}
                                        </p>
                                        {job.pin_comment && (
                                            <p className='mt-2 text-[10px] font-medium text-fuchsia-300 bg-fuchsia-500/5 p-2 rounded-lg border border-fuchsia-500/10 whitespace-pre-wrap'>
                                                {job.pin_comment}
                                            </p>
                                        )}
                                        {job.error && (
                                            <p className='mt-2 text-[10px] font-bold text-red-500 bg-red-500/5 p-2 rounded-lg border border-red-500/10'>
                                                {job.error}
                                            </p>
                                        )}
                                    </div>
                                </div>
                            ))
                        ) : (
                            <div className='h-full flex flex-col items-center justify-center text-center py-10 opacity-50 grayscale'>
                                <div className='text-4xl mb-4'>💤</div>
                                <p className='text-[10px] font-black text-zinc-500 uppercase tracking-[0.2em]'>
                                    Queue Empty
                                </p>
                            </div>
                        )}
                    </div>

                    <div className='mt-8 rounded-2xl border border-white/5 bg-black/30 p-5'>
                        <div className='flex items-center justify-between mb-4'>
                            <div>
                                <p className='text-[10px] font-black uppercase tracking-widest text-zinc-500'>
                                    Publish History
                                </p>
                                <h4 className='text-lg font-black text-white'>
                                    Distribution Jobs
                                </h4>
                            </div>
                            <span className='text-[10px] font-bold text-zinc-500'>
                                {selectedJobId
                                    ? selectedJobId
                                    : "No job selected"}
                            </span>
                        </div>

                        {distributionJobs.length > 0 ? (
                            <div className='space-y-3 max-h-56 overflow-auto pr-1'>
                                {distributionJobs.map((dist) => (
                                    <div
                                        key={dist.id}
                                        className='rounded-2xl border border-white/5 bg-black/40 p-4'
                                    >
                                        <div className='flex items-center justify-between gap-3 mb-2'>
                                            <div>
                                                <p className='text-sm font-black text-white uppercase tracking-tight'>
                                                    {dist.platform}
                                                </p>
                                                <p className='text-[10px] font-mono text-zinc-500'>
                                                    acct {dist.account_id}
                                                </p>
                                            </div>
                                            <span
                                                className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest ${dist.status === "completed" ? "bg-emerald-500/15 text-emerald-300 border-emerald-500/30" : dist.status === "failed" ? "bg-red-500/15 text-red-300 border-red-500/30" : "bg-amber-500/15 text-amber-300 border-amber-500/30"}`}
                                            >
                                                {dist.status}
                                            </span>
                                        </div>
                                        <div className='space-y-1'>
                                            <p className='text-[10px] text-zinc-500 font-bold uppercase tracking-widest'>
                                                Stage
                                            </p>
                                            <p className='text-xs text-zinc-300 break-all'>
                                                {dist.status_detail || "-"}
                                            </p>
                                            {dist.scheduled_at && (
                                                <>
                                                    <p className='text-[10px] text-zinc-500 font-bold uppercase tracking-widest mt-2'>
                                                        Scheduled
                                                    </p>
                                                    <p className='text-xs text-zinc-300 break-all'>
                                                        {formatTime(
                                                            dist.scheduled_at,
                                                        )}
                                                    </p>
                                                </>
                                            )}
                                            <p className='text-[10px] text-zinc-500 font-bold uppercase tracking-widest'>
                                                External ID
                                            </p>
                                            <p className='text-xs text-zinc-300 break-all'>
                                                {dist.external_id || "-"}
                                            </p>
                                            {dist.error && (
                                                <p className='text-xs text-red-400 break-all'>
                                                    {dist.error}
                                                </p>
                                            )}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className='text-xs text-zinc-500'>
                                No publish history for this job yet.
                            </p>
                        )}
                    </div>
                </div>
            </section>

            <div className='border-l-4 border-emerald-500 pl-6'>
                <h2 className='text-4xl font-black text-white tracking-tighter uppercase font-mono'>
                    READY FOR REVIEW
                </h2>
                <p className='text-zinc-500 mt-2 font-medium'>
                    Verify AI outputs before final distribution.
                </p>
            </div>

            {loading ? (
                <div className='flex justify-center py-32'>
                    <div className='animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500 shadow-[0_0_20px_rgba(16,185,129,0.3)]'></div>
                </div>
            ) : (
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-8'>
                    {clips && clips.length > 0 ? (
                        clips.map((clip) => (
                            <div
                                key={clip.id}
                                className='bg-card rounded-[2rem] overflow-hidden border border-white/5 shadow-2xl flex flex-col transition-all duration-500 hover:border-emerald-500/50 hover:-translate-y-2 group/card'
                            >
                                <div className='aspect-[9/16] bg-black relative overflow-hidden'>
                                    <video
                                        src={clip.s3_path}
                                        controls
                                        className='w-full h-full object-contain relative z-10'
                                    />

                                    {/* Floating Viral Score */}
                                    <div className='absolute top-4 right-4 z-20 bg-black/80 backdrop-blur-xl text-emerald-400 px-4 py-2 rounded-2xl text-[10px] font-black border border-emerald-500/30 shadow-2xl tracking-[0.1em]'>
                                        V-SCORE {clip.viral_score}%
                                    </div>
                                </div>

                                <div className='p-6 flex-1 flex flex-col'>
                                    <h3 className='text-lg font-black text-white mb-2 line-clamp-2 leading-tight uppercase group-hover/card:text-emerald-400 transition-colors'>
                                        {clip.headline}
                                    </h3>
                                    <div className='flex items-center gap-2 mb-6'>
                                        <span className='text-[9px] font-mono font-bold text-zinc-600 bg-black/40 px-2 py-1 rounded-md border border-white/5 uppercase tracking-tighter'>
                                            TC {clip.start_time} -{" "}
                                            {clip.end_time}
                                        </span>
                                    </div>

                                    <div className='mt-auto space-y-4'>
                                        <div className='flex items-center justify-between'>
                                            <span className='text-[10px] font-black text-zinc-600 uppercase tracking-widest'>
                                                STATUS
                                            </span>
                                            <span
                                                className={`text-[10px] font-black px-3 py-1 rounded-full border shadow-sm tracking-[0.1em] ${
                                                    clip.status === "approved"
                                                        ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-400"
                                                        : clip.status ===
                                                            "rejected"
                                                          ? "bg-red-500/10 border-red-500/20 text-red-400"
                                                          : "bg-amber-500/10 border-amber-500/20 text-amber-400"
                                                }`}
                                            >
                                                {(
                                                    clip.status || "pending"
                                                ).toUpperCase()}
                                            </span>
                                        </div>

                                        <div className='grid grid-cols-2 gap-3'>
                                            <button
                                                onClick={() =>
                                                    updateStatus(
                                                        clip.id,
                                                        "approved",
                                                    )
                                                }
                                                disabled={
                                                    clip.status === "approved"
                                                }
                                                className={`font-black text-[11px] uppercase tracking-widest py-3.5 rounded-xl transition-all ${
                                                    clip.status === "approved"
                                                        ? "bg-zinc-900 text-zinc-700 cursor-not-allowed border border-white/5"
                                                        : "bg-emerald-600 hover:bg-emerald-500 text-white shadow-lg shadow-emerald-900/20 active:scale-95"
                                                }`}
                                            >
                                                Approve
                                            </button>
                                            <button
                                                onClick={() =>
                                                    updateStatus(
                                                        clip.id,
                                                        "rejected",
                                                    )
                                                }
                                                disabled={
                                                    clip.status === "rejected"
                                                }
                                                className={`font-black text-[11px] uppercase tracking-widest py-3.5 rounded-xl transition-all border ${
                                                    clip.status === "rejected"
                                                        ? "bg-zinc-900 text-zinc-700 border-white/5 cursor-not-allowed"
                                                        : "bg-zinc-800 hover:bg-red-900/20 hover:text-red-400 text-white border-white/5 active:scale-95"
                                                }`}
                                            >
                                                Reject
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className='col-span-full text-center py-40 bg-card rounded-[3rem] border-2 border-dashed border-white/5'>
                            <div className='text-7xl mb-6 opacity-30 grayscale'>
                                🎬
                            </div>
                            <p className='text-zinc-500 text-xl font-black uppercase tracking-[0.2em]'>
                                Depleted Queue
                            </p>
                            <p className='text-zinc-600 mt-2 font-medium'>
                                Initiate pipeline to generate new content.
                            </p>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
