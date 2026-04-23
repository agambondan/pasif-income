'use client';

import { useEffect, useState } from 'react';
import AccountSelector, {
    type ConnectedAccount,
} from '@/components/account-selector';
import {
    DashboardSurfaceHeader,
} from '@/components/dashboard-surface';

export default function VideoClipper() {
    const [videoUrl, setVideoUrl] = useState("");
    const [isProcessing, setIsProcessing] = useState(false);
    const [status, setStatus] = useState<string | null>(null);
    const [accounts, setAccounts] = useState<ConnectedAccount[]>([]);
    const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);

    useEffect(() => {
        let cancelled = false;

        const fetchAccounts = async () => {
            try {
                const res = await fetch(`/api/accounts`, { credentials: "include" });
                if (!res.ok) {
                    return;
                }
                const data = await res.json();
                if (!cancelled) {
                    setAccounts(data || []);
                }
            } catch (err) {
                console.error("Failed to fetch accounts:", err);
            }
        };

        void fetchAccounts();

        return () => {
            cancelled = true;
        };
    }, []);

    const browserReadyAccounts = accounts.filter(
        (account) =>
            account.auth_method !== 'chromium_profile' ||
            account.browser_status === 'ready',
    );

    const startClipping = async () => {
        try {
            if (!videoUrl) {
                setStatus("Video URL is required");
                return;
            }
            if (selectedAccounts.length === 0) {
                setStatus("Select at least one destination account");
                return;
            }

            setIsProcessing(true);
            setStatus(null);

            const destinations = selectedAccounts.map((id) => {
                const acc = accounts.find((a) => a.id === id);
                return {
                    platform: acc?.platform_id || "unknown",
                    account_id: id,
                };
            });

            const res = await fetch(`/api/generate`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ 
                    niche: "clipping", 
                    topic: "podcast-factory",
                    video_url: videoUrl,
                    destinations
                }),
            });

            if (res.ok) {
                const data = await res.json();
                setStatus(`Clipping job ${data.id} initiated. AI is analyzing segments for ${destinations.length} destinations.`);
                setVideoUrl("");
                setSelectedAccounts([]);
            } else {
                const errText = await res.text();
                setStatus(`Failed: ${errText || "Unknown error"}`);
            }
        } catch (err) {
            console.error("Clipping error:", err);
            setStatus("Error connecting to production pipeline.");
        } finally {
            setIsProcessing(false);
        }
    };

    return (
        <div className='space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-10'>
            <DashboardSurfaceHeader
                eyebrow='Clipper Portal'
                title='Podcast clips factory'
                description='Turn a source URL or backend-visible file path into a reviewable clipping job with clear destination selection.'
                actions={
                    <>
                        <div className="rounded-2xl border border-white/5 bg-black/30 px-5 py-3 text-[10px] font-black uppercase tracking-widest text-zinc-500">
                            {browserReadyAccounts.length}/{accounts.length || 0} destination accounts ready
                        </div>
                        <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 px-5 py-3 text-[10px] font-black uppercase tracking-widest text-emerald-300">
                            Source + destination locked to dashboard session
                        </div>
                    </>
                }
            />

            <section className="grid gap-4 md:grid-cols-3">
                <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
                    <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Connected Accounts</p>
                    <p className="mt-2 text-3xl font-black text-white">{accounts.length}</p>
                </div>
                <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
                    <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Browser Ready</p>
                    <p className="mt-2 text-3xl font-black text-emerald-400">{browserReadyAccounts.length}</p>
                </div>
                <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
                    <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Selected Destinations</p>
                    <p className="mt-2 text-3xl font-black text-amber-400">{selectedAccounts.length}</p>
                </div>
            </section>

            <div className='w-full'>
                <div className='bg-card border border-white/5 rounded-[3rem] p-12 shadow-2xl relative overflow-hidden group'>
                    <div className='absolute -top-24 -right-24 w-96 h-96 bg-emerald-500/5 rounded-full blur-3xl group-hover:bg-emerald-500/10 transition-colors'></div>
                    
                    <div className='relative z-10'>
                        <div className='space-y-10'>
                            <div className='space-y-3'>
                                <label className='text-[10px] font-black text-zinc-500 uppercase tracking-[0.2em] ml-2'>Video Source URL</label>
                                <input
                                    value={videoUrl}
                                    onChange={(e) => setVideoUrl(e.target.value)}
                                    className='w-full rounded-[1.5rem] border border-white/10 bg-black/40 px-8 py-6 text-white font-bold outline-none transition-all focus:border-emerald-500 focus:ring-8 focus:ring-emerald-500/5 placeholder:text-zinc-700'
                                    placeholder='Paste YouTube link or backend-visible file path...'
                                />
                                <div className='grid gap-3 rounded-2xl border border-white/5 bg-black/20 px-4 py-4 md:grid-cols-3'>
                                    <p className='text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        Accepted sources: YouTube URL or direct file path accessible from the backend.
                                    </p>
                                    <p className='text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        Destination accounts must be connected first, then selected below.
                                    </p>
                                    <p className='text-[10px] font-bold text-zinc-500 uppercase tracking-widest'>
                                        Browser profile destinations are gated until the profile reaches READY.
                                    </p>
                                </div>
                            </div>

                            {/* Distribution Matrix Section */}
                            <div className='space-y-4'>
                                <AccountSelector
                                    title='Target Distribution Matrix'
                                    subtitle='Choose the destination accounts that will receive the clip job.'
                                    selectionHint='Only ready Chromium profiles and API accounts can be selected.'
                                    accounts={accounts}
                                    selectedIds={selectedAccounts}
                                    onChange={setSelectedAccounts}
                                    emptyMessage='No connected accounts found. Go to Integrations first.'
                                />
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
                                <div className={`mt-6 rounded-2xl px-6 py-4 text-xs font-bold text-center uppercase tracking-widest ${status.toLowerCase().includes('failed') || status.toLowerCase().includes('error') ? 'border border-red-500/20 bg-red-500/10 text-red-300' : 'border border-emerald-500/20 bg-emerald-500/10 text-emerald-400'}`}>
                                    {status}
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
