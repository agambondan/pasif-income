'use client';

import { useEffect, useState, useCallback } from 'react';
import { Clip } from '@/types/clip';

type GenerationJob = {
  id: string;
  niche: string;
  topic: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
  error?: string;
  created_at: string;
  updated_at: string;
};

const API_BASE_URL = '';

const jobStatusStyles: Record<GenerationJob['status'], string> = {
  queued: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
  running: 'bg-blue-500/15 text-blue-300 border-blue-500/30',
  completed: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
  failed: 'bg-red-500/15 text-red-300 border-red-500/30',
};

function formatTime(value: string) {
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
  const [lastUpdated, setLastUpdated] = useState('Never');
  const [niche, setNiche] = useState('stoicism');
  const [topic, setTopic] = useState('how to control your mind');
  const [accounts, setAccounts] = useState<any[]>([]);
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);

  const fetchAccounts = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/accounts`);
      if (res.ok) {
        const data = await res.json();
        setAccounts(data || []);
      }
    } catch (err) {
      console.error('Failed to fetch accounts:', err);
    }
  }, []);

  const fetchClips = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/clips`);
      if (!res.ok) {
        console.error(`Failed to fetch clips: ${res.status}`);
        return;
      }
      const data = await res.json();
      setClips(data || []);
    } catch (err) {
      console.error('Failed to fetch clips:', err);
    }
  }, []);

  const fetchJobs = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/jobs`);
      if (!res.ok) {
        console.error(`Failed to fetch jobs: ${res.status}`);
        return;
      }
      const data = await res.json();
      setJobs(data || []);
    } catch (err) {
      console.error('Failed to fetch jobs:', err);
    }
  }, []);

  const fetchBackendHealth = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/health`);
      setBackendOnline(res.ok);
    } catch (err) {
      console.error('Failed to fetch backend health:', err);
      setBackendOnline(false);
    }
  }, []);

  const refreshAll = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchBackendHealth(), fetchClips(), fetchJobs(), fetchAccounts()]);
    setLoading(false);
    setLastUpdated(new Date().toLocaleString());
  }, [fetchBackendHealth, fetchClips, fetchJobs, fetchAccounts]);

  const pollState = useCallback(async () => {
    await Promise.all([fetchBackendHealth(), fetchClips(), fetchJobs(), fetchAccounts()]);
    setLastUpdated(new Date().toLocaleString());
  }, [fetchBackendHealth, fetchClips, fetchJobs, fetchAccounts]);

  useEffect(() => {
    Promise.resolve().then(() => refreshAll());
    const interval = window.setInterval(() => {
      void pollState();
    }, 10000);

    return () => window.clearInterval(interval);
  }, [refreshAll, pollState]);

  const updateStatus = async (id: string, status: Clip['status']) => {
    try {
      await fetch(`${API_BASE_URL}/api/clips`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, status }),
      });
      await fetchClips();
    } catch (err) {
      console.error('Failed to update status:', err);
    }
  };

  const startGeneration = async () => {
    try {
      setIsGenerating(true);
      setStatusMessage(null);

      const destinations = selectedAccounts.map(id => {
        const acc = accounts.find(a => a.id === id);
        return { platform: acc.platform_id, account_id: id };
      });

      const res = await fetch(`${API_BASE_URL}/api/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ niche, topic, destinations }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `HTTP ${res.status}`);
      }
      const job = await res.json();
      setStatusMessage(`Job ${job.id} queued for ${job.niche}: ${job.topic}`);
      await Promise.all([fetchClips(), fetchJobs()]);
    } catch (err) {
      console.error('Failed to start generation:', err);
      setStatusMessage(err instanceof Error ? err.message : 'Failed to start generation');
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div className="flex flex-col md:flex-row justify-between items-end gap-6 border-l-4 border-blue-500 pl-6">
        <div>
           <h2 className="text-4xl font-black text-white tracking-tighter uppercase">Operations</h2>
           <p className="text-zinc-500 mt-2 font-medium">Control the AI production pipeline and monitor live jobs.</p>
        </div>
        <div className="flex gap-4">
            <button 
                onClick={refreshAll}
                className="bg-zinc-900 hover:bg-zinc-800 px-6 py-3 rounded-2xl border border-white/5 text-xs font-bold transition-all active:scale-95 shadow-xl"
            >
                REFRESH DATA
            </button>
            <div className={`px-6 py-3 rounded-2xl border text-xs font-black tracking-widest ${backendOnline ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' : 'bg-red-500/10 border-red-500/20 text-red-400'}`}>
                {backendOnline ? '● BACKEND ONLINE' : '● BACKEND OFFLINE'}
            </div>
        </div>
      </div>

      <section className="grid gap-8 lg:grid-cols-[1.4fr_0.6fr]">
        <div className="bg-card border border-white/5 rounded-[2.5rem] p-10 shadow-2xl relative overflow-hidden group">
          <div className="absolute -top-24 -right-24 w-64 h-64 bg-blue-500/5 rounded-full blur-3xl group-hover:bg-blue-500/10 transition-colors"></div>
          
          <div className="relative z-10">
            <div className="flex items-center gap-3 mb-8">
               <div className="w-10 h-10 bg-blue-500/20 rounded-xl flex items-center justify-center text-xl shadow-lg border border-blue-500/30 text-blue-400">⚡</div>
               <h3 className="text-2xl font-bold text-white uppercase tracking-tight">New Production Job</h3>
            </div>
            
            <div className="grid gap-6 md:grid-cols-2 mb-8">
              <div className="space-y-2">
                <label className="text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1">Niche Architecture</label>
                <input
                  value={niche}
                  onChange={(e) => setNiche(e.target.value)}
                  className="w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10"
                  placeholder="stoicism"
                />
              </div>
              <div className="space-y-2">
                <label className="text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1">Content Concept</label>
                <input
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                  className="w-full rounded-2xl border border-white/10 bg-black/40 px-6 py-4 text-white font-bold outline-none transition-all focus:border-blue-500 focus:ring-4 focus:ring-blue-500/10"
                  placeholder="how to control your mind"
                />
              </div>
            </div>

            {accounts.length > 0 && (
              <div className="mb-10">
                <span className="text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1 block mb-4 text-center">Distribution Matrix</span>
                <div className="flex flex-wrap justify-center gap-3">
                  {accounts.map(acc => (
                    <label key={acc.id} className={`flex items-center gap-3 border px-5 py-3 rounded-2xl cursor-pointer transition-all duration-300 ${selectedAccounts.includes(acc.id) ? 'bg-emerald-500/10 border-emerald-500/50 shadow-lg shadow-emerald-500/5' : 'bg-black/40 border-white/5 hover:border-white/20 hover:bg-black/60'}`}>
                      <input 
                        type="checkbox" 
                        className="hidden"
                        checked={selectedAccounts.includes(acc.id)}
                        onChange={(e) => {
                          if (e.target.checked) setSelectedAccounts([...selectedAccounts, acc.id]);
                          else setSelectedAccounts(selectedAccounts.filter(id => id !== acc.id));
                        }}
                      />
                      <div className={`w-5 h-5 rounded-lg border-2 flex items-center justify-center transition-all ${selectedAccounts.includes(acc.id) ? 'bg-emerald-500 border-emerald-500 rotate-0' : 'border-zinc-700 rotate-45'}`}>
                        {selectedAccounts.includes(acc.id) && <span className="text-black text-xs font-black">✓</span>}
                      </div>
                      <div className="flex flex-col">
                        <span className="text-sm font-bold text-white">{acc.display_name}</span>
                        <span className="text-[9px] text-zinc-500 font-black uppercase tracking-widest">{acc.platform_id}</span>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            )}

            <button
              onClick={startGeneration}
              disabled={isGenerating}
              className="w-full rounded-2xl bg-gradient-to-r from-blue-600 to-blue-500 py-5 font-black text-white text-sm uppercase tracking-[0.2em] transition-all hover:from-blue-500 hover:to-blue-400 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed shadow-xl shadow-blue-900/20 flex items-center justify-center gap-3"
            >
              {isGenerating ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-2 border-white/30 border-t-white"></div>
                  INITIALIZING...
                </>
              ) : 'EXECUTE PRODUCTION'}
            </button>
            
            {statusMessage && (
              <div className="mt-6 rounded-2xl border border-blue-500/20 bg-blue-500/10 px-6 py-4 text-xs font-bold text-blue-400 text-center animate-bounce">
                {statusMessage}
              </div>
            )}
          </div>
        </div>

        <div className="bg-card border border-white/5 rounded-[2.5rem] p-8 shadow-2xl flex flex-col">
          <div className="flex items-center gap-3 mb-8">
             <div className="w-10 h-10 bg-emerald-500/20 rounded-xl flex items-center justify-center text-xl shadow-lg border border-emerald-500/30 text-emerald-400">📊</div>
             <h3 className="text-2xl font-bold text-white uppercase tracking-tight">Active Jobs</h3>
          </div>
          
          <div className="space-y-4 flex-1 overflow-auto max-h-[500px] pr-2">
            {jobs && jobs.length > 0 ? jobs.map((job) => (
              <div key={job.id} className="rounded-2xl border border-white/5 bg-black/40 p-5 hover:border-white/20 transition-all group/job">
                <div className="flex items-start justify-between gap-4 mb-3">
                  <div>
                    <p className="font-black text-white text-sm group-hover:text-blue-400 transition-colors uppercase tracking-tight">{job.niche}</p>
                    <p className="text-xs text-zinc-500 font-medium line-clamp-1 mt-1 uppercase tracking-widest">{job.topic}</p>
                  </div>
                  <span className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest shadow-sm ${jobStatusStyles[job.status]}`}>
                    {job.status}
                  </span>
                </div>
                <div className="flex flex-col gap-1">
                    <p className="text-[10px] text-zinc-600 font-bold uppercase tracking-tighter">
                    {formatTime(job.created_at)}
                    </p>
                    {job.error && <p className="mt-2 text-[10px] font-bold text-red-500 bg-red-500/5 p-2 rounded-lg border border-red-500/10">{job.error}</p>}
                </div>
              </div>
            )) : (
              <div className="h-full flex flex-col items-center justify-center text-center py-10 opacity-50 grayscale">
                <div className="text-4xl mb-4">💤</div>
                <p className="text-[10px] font-black text-zinc-500 uppercase tracking-[0.2em]">Queue Empty</p>
              </div>
            )}
          </div>
        </div>
      </section>

      <div className="border-l-4 border-emerald-500 pl-6">
        <h2 className="text-4xl font-black text-white tracking-tighter uppercase font-mono">READY FOR REVIEW</h2>
        <p className="text-zinc-500 mt-2 font-medium">Verify AI outputs before final distribution.</p>
      </div>

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500 shadow-[0_0_20px_rgba(16,185,129,0.3)]"></div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-8">
          {clips && clips.length > 0 ? (
            clips.map((clip) => (
              <div key={clip.id} className="bg-card rounded-[2rem] overflow-hidden border border-white/5 shadow-2xl flex flex-col transition-all duration-500 hover:border-emerald-500/50 hover:-translate-y-2 group/card">
                <div className="aspect-[9/16] bg-black relative overflow-hidden">
                  <video 
                    src={clip.s3_path} 
                    controls 
                    className="w-full h-full object-contain relative z-10"
                  />
                  
                  {/* Floating Viral Score */}
                  <div className="absolute top-4 right-4 z-20 bg-black/80 backdrop-blur-xl text-emerald-400 px-4 py-2 rounded-2xl text-[10px] font-black border border-emerald-500/30 shadow-2xl tracking-[0.1em]">
                    V-SCORE {clip.viral_score}%
                  </div>
                </div>

                <div className="p-6 flex-1 flex flex-col">
                  <h3 className="text-lg font-black text-white mb-2 line-clamp-2 leading-tight uppercase group-hover/card:text-emerald-400 transition-colors">{clip.headline}</h3>
                  <div className="flex items-center gap-2 mb-6">
                    <span className="text-[9px] font-mono font-bold text-zinc-600 bg-black/40 px-2 py-1 rounded-md border border-white/5 uppercase tracking-tighter">
                        TC {clip.start_time} - {clip.end_time}
                    </span>
                  </div>
                  
                  <div className="mt-auto space-y-4">
                    <div className="flex items-center justify-between">
                        <span className="text-[10px] font-black text-zinc-600 uppercase tracking-widest">STATUS</span>
                        <span className={`text-[10px] font-black px-3 py-1 rounded-full border shadow-sm tracking-[0.1em] ${
                            clip.status === 'approved' ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400' : 
                            clip.status === 'rejected' ? 'bg-red-500/10 border-red-500/20 text-red-400' : 'bg-amber-500/10 border-amber-500/20 text-amber-400'
                        }`}>
                            {(clip.status || 'pending').toUpperCase()}
                        </span>
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                        <button 
                        onClick={() => updateStatus(clip.id, 'approved')}
                        disabled={clip.status === 'approved'}
                        className={`font-black text-[11px] uppercase tracking-widest py-3.5 rounded-xl transition-all ${
                            clip.status === 'approved' 
                            ? 'bg-zinc-900 text-zinc-700 cursor-not-allowed border border-white/5' 
                            : 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-lg shadow-emerald-900/20 active:scale-95'
                        }`}
                        >
                        Approve
                        </button>
                        <button 
                        onClick={() => updateStatus(clip.id, 'rejected')}
                        disabled={clip.status === 'rejected'}
                        className={`font-black text-[11px] uppercase tracking-widest py-3.5 rounded-xl transition-all border ${
                            clip.status === 'rejected'
                            ? 'bg-zinc-900 text-zinc-700 border-white/5 cursor-not-allowed'
                            : 'bg-zinc-800 hover:bg-red-900/20 hover:text-red-400 text-white border-white/5 active:scale-95'
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
            <div className="col-span-full text-center py-40 bg-card rounded-[3rem] border-2 border-dashed border-white/5">
              <div className="text-7xl mb-6 opacity-30 grayscale">🎬</div>
              <p className="text-zinc-500 text-xl font-black uppercase tracking-[0.2em]">Depleted Queue</p>
              <p className="text-zinc-600 mt-2 font-medium">Initiate pipeline to generate new content.</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
