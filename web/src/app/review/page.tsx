'use client';

import { useEffect, useState, useCallback } from 'react';
import { Clip } from '@/types/clip';

export default function ReviewQueue() {
    const [clips, setClips] = useState<Clip[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchClips = useCallback(async () => {
        try {
            const res = await fetch(`/api/clips`, { credentials: "include" });
            if (res.ok) {
                const data = await res.json();
                setClips(data || []);
            }
        } catch (err) {
            console.error('Failed to fetch clips:', err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        const timer = window.setTimeout(() => {
            void fetchClips();
        }, 0);
        return () => window.clearTimeout(timer);
    }, [fetchClips]);

    const updateStatus = async (id: string, status: Clip['status']) => {
        try {
            await fetch(`/api/clips`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({ id, status }),
            });
            await fetchClips();
        } catch (err) {
            console.error('Failed to update status:', err);
        }
    };

    return (
        <div className='space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700'>
            <div className='border-l-4 border-amber-500 pl-6'>
                <h2 className='text-4xl font-black text-white tracking-tighter uppercase font-mono'>
                    Review Queue
                </h2>
                <p className='text-zinc-500 mt-2 font-medium'>
                    Quality Control center. Approve or reject AI generated clips before distribution.
                </p>
            </div>

            {loading ? (
                <div className='flex justify-center py-32'>
                    <div className='animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500 shadow-[0_0_20px_rgba(251,191,36,0.3)]'></div>
                </div>
            ) : (
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-8'>
                    {clips && clips.length > 0 ? (
                        clips.map((clip) => (
                            <div key={clip.id} className='bg-card rounded-[2rem] overflow-hidden border border-white/5 shadow-2xl flex flex-col transition-all duration-500 hover:border-amber-500/50 hover:-translate-y-2 group/card'>
                                <div className='aspect-[9/16] bg-black relative overflow-hidden'>
                                    <video src={clip.s3_path} controls className='w-full h-full object-contain relative z-10' />
                                    <div className='absolute top-4 right-4 z-20 bg-black/80 backdrop-blur-xl text-amber-400 px-4 py-2 rounded-2xl text-[10px] font-black border border-amber-500/30 shadow-2xl tracking-[0.1em]'>
                                        V-SCORE {clip.viral_score}%
                                    </div>
                                </div>
                                <div className='p-6 flex-1 flex flex-col'>
                                    <h3 className='text-lg font-black text-white mb-2 line-clamp-2 leading-tight uppercase group-hover/card:text-amber-400 transition-colors'>
                                        {clip.headline}
                                    </h3>
                                    <div className='flex items-center gap-2 mb-6'>
                                        <span className='text-[9px] font-mono font-bold text-zinc-600 bg-black/40 px-2 py-1 rounded-md border border-white/5 uppercase tracking-tighter'>
                                            TC {clip.start_time} - {clip.end_time}
                                        </span>
                                    </div>
                                    <div className='mt-auto space-y-4'>
                                        <div className='flex items-center justify-between'>
                                            <span className='text-[10px] font-black text-zinc-600 uppercase tracking-widest'>STATUS</span>
                                            <span className={`text-[10px] font-black px-3 py-1 rounded-full border shadow-sm tracking-[0.1em] ${clip.status === 'approved' ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' : clip.status === 'rejected' ? 'bg-red-500/10 border-red-500/20 text-red-400' : 'bg-amber-500/10 border-amber-500/20 text-amber-400'}`}>
                                                {(clip.status || 'pending').toUpperCase()}
                                            </span>
                                        </div>
                                        <div className='grid grid-cols-2 gap-3'>
                                            <button onClick={() => updateStatus(clip.id, 'approved')} disabled={clip.status === 'approved'} className={`font-black text-[11px] uppercase tracking-widest py-3.5 rounded-xl transition-all ${clip.status === 'approved' ? 'bg-zinc-900 text-zinc-700 cursor-not-allowed border border-white/5' : 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-lg shadow-emerald-900/20 active:scale-95'}`}>Approve</button>
                                            <button onClick={() => updateStatus(clip.id, 'rejected')} disabled={clip.status === 'rejected'} className={`font-black text-[11px] uppercase tracking-widest py-3.5 rounded-xl transition-all border ${clip.status === 'rejected' ? 'bg-zinc-900 text-zinc-700 border-white/5 cursor-not-allowed' : 'bg-zinc-800 hover:bg-red-900/20 hover:text-red-400 text-white border-white/5 active:scale-95'}`}>Reject</button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        ))
                    ) : (
                        <div className='col-span-full text-center py-40 bg-card rounded-[3rem] border-2 border-dashed border-white/5'>
                            <div className='text-7xl mb-6 opacity-30 grayscale'>💤</div>
                            <p className='text-zinc-500 text-xl font-black uppercase tracking-[0.2em]'>Review Queue Empty</p>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
