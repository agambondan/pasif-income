'use client';

import { useCallback, useEffect, useState } from 'react';

type ClipJob = {
    id: string;
    niche: string;
    topic: string;
    video_url?: string;
    status: 'queued' | 'running' | 'completed' | 'failed';
    error?: string;
    created_at?: string;
    updated_at?: string;
};

export default function VideoClipper() {
    const [videoUrl, setVideoUrl] = useState("");
    const [isProcessing, setIsProcessing] = useState(false);
    const [status, setStatus] = useState<string | null>(null);
    const [job, setJob] = useState<ClipJob | null>(null);

    const refreshJob = useCallback(async (jobId: string) => {
        const res = await fetch(`/api/jobs/${jobId}`, {
            credentials: "include",
        });
        if (!res.ok) {
            return null;
        }
        const data = (await res.json()) as ClipJob;
        setJob(data);
        if (data.status === "running") {
            setStatus(`Clipping job ${data.id} is processing the footage...`);
        } else if (data.status === "completed") {
            setStatus(`Clipping job ${data.id} completed. Check the review queue.`);
        } else if (data.status === "failed") {
            setStatus(
                `Clipping job ${data.id} failed${data.error ? `: ${data.error}` : ""}`,
            );
        } else {
            setStatus(`Clipping job ${data.id} queued.`);
        }
        return data;
    }, []);

    useEffect(() => {
        if (!job?.id) {
            return;
        }

        let stopped = false;
        const tick = async () => {
            if (stopped) {
                return;
            }
            const next = await refreshJob(job.id);
            if (stopped || !next) {
                return;
            }
            if (next.status === "completed" || next.status === "failed") {
                stopped = true;
            }
        };

        void tick();
        const timer = window.setInterval(() => {
            void tick();
        }, 5000);

        return () => {
            stopped = true;
            window.clearInterval(timer);
        };
    }, [job?.id, refreshJob]);

    const startClipping = async () => {
        try {
            if (!videoUrl) {
                setStatus("Video URL is required");
                return;
            }
            setIsProcessing(true);
            setStatus(null);
            setJob(null);

            const res = await fetch(`/api/generate`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    niche: "clipping",
                    topic: "podcast-factory",
                    video_url: videoUrl,
                }),
            });

            if (!res.ok) {
                throw new Error(await res.text());
            }

            const data = (await res.json()) as ClipJob;
            setJob(data);
            setStatus(`Clipping job ${data.id} queued. Waiting for backend pipeline...`);
        } catch (err) {
            console.error("Clipping error:", err);
            setStatus(
                err instanceof Error
                    ? err.message
                    : "Error connecting to production pipeline.",
            );
        } finally {
            setIsProcessing(false);
        }
    };

    return (
        <div className='space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700'>
            <div className='border-l-4 border-emerald-500 pl-6'>
                <h2 className='text-4xl font-black text-white tracking-tighter uppercase'>Podcast Clips Factory</h2>
                <p className='text-zinc-500 mt-2 font-medium'>Transform long-form videos into viral vertical clips using Vision AI.</p>
            </div>

            <div className='max-w-4xl'>
                <div className='bg-card border border-white/5 rounded-[3rem] p-12 shadow-2xl relative overflow-hidden group'>
                    <div className='absolute -top-24 -right-24 w-96 h-96 bg-emerald-500/5 rounded-full blur-3xl group-hover:bg-emerald-500/10 transition-colors'></div>
                    
                    <div className='relative z-10'>
                        <div className='flex items-center gap-4 mb-10'>
                            <div className='w-14 h-14 bg-emerald-500/20 rounded-2xl flex items-center justify-center text-3xl shadow-lg border border-emerald-500/30 text-emerald-400'>🎬</div>
                            <div>
                                <h3 className='text-2xl font-bold text-white uppercase tracking-tight'>New Clipping Job</h3>
                                <p className='text-zinc-500 text-xs font-bold uppercase tracking-widest mt-1'>Vision AI Pipeline v4.0</p>
                            </div>
                        </div>

                        <div className='space-y-8'>
                            <div className='space-y-3'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-[0.2em] ml-2'>Video Source URL</label>
                                <input
                                    value={videoUrl}
                                    onChange={(e) => setVideoUrl(e.target.value)}
                                    className='w-full rounded-[1.5rem] border border-white/10 bg-black/40 px-8 py-6 text-white font-bold outline-none transition-all focus:border-emerald-500 focus:ring-8 focus:ring-emerald-500/5 placeholder:text-zinc-700'
                                    placeholder='Paste YouTube link or local storage path...'
                                />
                            </div>

                        <div className='grid grid-cols-1 md:grid-cols-3 gap-4'>
                            <div className='bg-zinc-900/50 p-6 rounded-2xl border border-white/5'>
                                <p className='text-[9px] font-black text-zinc-500 uppercase tracking-widest mb-2'>Analysis</p>
                                <p className='text-xs font-bold text-zinc-300 uppercase'>Backend Strategist</p>
                            </div>
                            <div className='bg-zinc-900/50 p-6 rounded-2xl border border-white/5'>
                                <p className='text-[9px] font-black text-zinc-500 uppercase tracking-widest mb-2'>Format</p>
                                <p className='text-xs font-bold text-zinc-300 uppercase'>Vertical Clips</p>
                            </div>
                            <div className='bg-zinc-900/50 p-6 rounded-2xl border border-white/5'>
                                <p className='text-[9px] font-black text-zinc-500 uppercase tracking-widest mb-2'>Audio</p>
                                <p className='text-xs font-bold text-zinc-300 uppercase'>Captioned Render</p>
                            </div>
                        </div>

                            <button
                                onClick={startClipping}
                                disabled={isProcessing}
                                className='w-full rounded-[1.5rem] bg-gradient-to-r from-emerald-600 to-teal-500 py-6 font-black text-white text-sm uppercase tracking-[0.3em] transition-all hover:from-emerald-500 hover:to-teal-400 active:scale-[0.98] disabled:opacity-50 shadow-2xl shadow-emerald-900/40 flex items-center justify-center gap-4'
                            >
                                {isProcessing ? (
                                    <>
                                        <div className='animate-spin rounded-full h-5 w-5 border-2 border-white/30 border-t-white'></div>
                                        PROCESSING FOOTAGE...
                                    </>
                                ) : (
                                    "INITIALIZE CLIPPING"
                                )}
                            </button>

                            {status && (
                                <div className='mt-6 rounded-2xl border border-emerald-500/20 bg-emerald-500/10 px-6 py-4 text-xs font-bold text-emerald-400 text-center animate-pulse uppercase tracking-widest'>
                                    {status}
                                </div>
                            )}

                            {job && (
                                <div className='rounded-2xl border border-white/5 bg-black/30 p-5'>
                                    <div className='flex items-center justify-between gap-3'>
                                        <div>
                                            <p className='text-[10px] font-black text-zinc-500 uppercase tracking-widest'>Current Job</p>
                                            <h4 className='text-sm font-bold text-white break-all'>{job.id}</h4>
                                        </div>
                                        <span className={`rounded-lg border px-3 py-1 text-[9px] font-black uppercase tracking-widest ${job.status === "completed" ? "bg-emerald-500/15 text-emerald-300 border-emerald-500/30" : job.status === "failed" ? "bg-red-500/15 text-red-300 border-red-500/30" : "bg-amber-500/15 text-amber-300 border-amber-500/30"}`}>
                                            {job.status}
                                        </span>
                                    </div>
                                    <div className='mt-4 space-y-2 text-xs text-zinc-400'>
                                        <p><span className='font-black uppercase tracking-widest text-zinc-500'>Source:</span> {job.video_url || videoUrl}</p>
                                        <p><span className='font-black uppercase tracking-widest text-zinc-500'>Topic:</span> {job.topic}</p>
                                        {job.error && (
                                            <p className='text-red-400 font-medium break-all'>{job.error}</p>
                                        )}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
