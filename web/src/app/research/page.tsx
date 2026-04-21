'use client';

import { useState } from 'react';

type IdeaSuggestion = {
    title: string;
    hook: string;
    angle: string;
    search_query: string;
    trend_source: string;
    score: number;
    reason: string;
};

export default function ResearchLab() {
    const [niche, setNiche] = useState("stoicism");
    const [isResearching, setIsResearching] = useState(false);
    const [ideaSuggestions, setIdeaSuggestions] = useState<IdeaSuggestion[]>([]);
    const [ideaWarnings, setIdeaWarnings] = useState<string[]>([]);
    const [statusMessage, setStatusMessage] = useState<string | null>(null);
    const [copiedQuery, setCopiedQuery] = useState<string | null>(null);

    const researchIdeas = async () => {
        try {
            setIsResearching(true);
            setStatusMessage(null);
            const res = await fetch(`/api/research/ideas`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ niche, limit: 5 }),
            });
            if (!res.ok) {
                throw new Error(await res.text());
            }
            const data = await res.json();
            setIdeaSuggestions(data.ideas || []);
            setIdeaWarnings(data.warnings || []);
            setStatusMessage(
                `Found ${data.ideas?.length || 0} ideas for ${data.niche || niche}`,
            );
        } catch (err) {
            console.error("Failed to fetch ideas:", err);
            setStatusMessage(
                err instanceof Error ? err.message : "Failed to fetch ideas",
            );
        } finally {
            setIsResearching(false);
        }
    };

    const copyQuery = async (query: string) => {
        try {
            await navigator.clipboard.writeText(query);
            setCopiedQuery(query);
            window.setTimeout(() => {
                setCopiedQuery((current) => (current === query ? null : current));
            }, 1500);
        } catch (err) {
            console.error("Failed to copy query:", err);
            setStatusMessage("Unable to copy query to clipboard");
        }
    };

    return (
        <div className='space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-10'>
            <div className='grid gap-8 lg:grid-cols-[0.8fr_2.2fr]'>
                <div className='bg-card border border-white/5 rounded-[2.5rem] p-10 shadow-2xl h-fit sticky top-24'>
                    <h3 className='text-xl font-bold text-white mb-6 uppercase tracking-tight'>Target Niche</h3>
                    <div className='space-y-4'>
                        <input
                            value={niche}
                            onChange={(e) => setNiche(e.target.value)}
                            className='w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-fuchsia-500 focus:ring-4 focus:ring-fuchsia-500/10'
                            placeholder='e.g. stoicism, crypto, horror'
                        />
                        <button
                            onClick={researchIdeas}
                            disabled={isResearching}
                            className='w-full rounded-2xl bg-fuchsia-600 hover:bg-fuchsia-500 py-4 font-black text-white text-xs uppercase tracking-widest transition-all active:scale-95 disabled:opacity-50'
                        >
                            {isResearching ? "ANALYZING SIGNALS..." : "START RESEARCH"}
                        </button>
                        {statusMessage && (
                            <p className='text-xs font-bold text-zinc-400 leading-relaxed'>
                                {statusMessage}
                            </p>
                        )}
                    </div>
                </div>

                <div className='space-y-6'>
                    {ideaWarnings.length > 0 && (
                        <div className='flex flex-wrap gap-2'>
                            {ideaWarnings.map((warning) => (
                                <span key={warning} className='rounded-full border border-amber-500/20 bg-amber-500/10 px-4 py-1.5 text-[10px] font-bold uppercase tracking-widest text-amber-300'>
                                    ⚠️ {warning}
                                </span>
                            ))}
                        </div>
                    )}

                    {ideaSuggestions.length > 0 ? (
                        <div className='grid gap-4'>
                            {ideaSuggestions.map((idea) => (
                                <div key={idea.title} className='bg-card border border-white/5 rounded-3xl p-8 hover:border-fuchsia-500/30 transition-all group relative overflow-hidden'>
                                    <div className='absolute -top-12 -right-12 w-32 h-32 bg-fuchsia-500/5 rounded-full blur-2xl group-hover:bg-fuchsia-500/10 transition-colors'></div>
                                    <div className='relative z-10'>
                                        <div className='flex justify-between items-start mb-4'>
                                            <div>
                                                <h4 className='text-xl font-black text-white uppercase tracking-tight'>{idea.title}</h4>
                                                <p className='text-[10px] font-mono text-fuchsia-400 mt-1 uppercase tracking-widest'>{idea.trend_source} · V-SCORE {idea.score}</p>
                                            </div>
                                            <button
                                                onClick={() => copyQuery(idea.search_query)}
                                                className='bg-white text-black px-4 py-2 rounded-xl text-[10px] font-black uppercase tracking-widest hover:bg-fuchsia-500 hover:text-white transition-all'
                                            >
                                                {copiedQuery === idea.search_query ? "Copied" : "Copy Query"}
                                            </button>
                                        </div>
                                        <p className='text-zinc-300 text-sm leading-relaxed mb-4 italic'>&ldquo;{idea.hook}&rdquo;</p>
                                        <div className='bg-black/40 p-4 rounded-2xl border border-white/5'>
                                            <p className='text-[10px] font-black text-zinc-500 uppercase tracking-widest mb-2'>AI Angle</p>
                                            <p className='text-xs text-zinc-400 font-medium'>{idea.angle}</p>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    ) : (
                        <div className='border-2 border-dashed border-white/5 rounded-[2.5rem] py-32 text-center'>
                             <div className='text-6xl mb-6 opacity-20 grayscale'>🔬</div>
                             <p className='text-zinc-600 font-bold uppercase tracking-widest'>Laboratory Idle. Input niche to start.</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
