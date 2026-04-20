'use client';

import { useEffect, useState } from 'react';

type DistributionJob = {
  id: number;
  generation_job_id: string;
  account_id: string;
  platform: string;
  status: 'pending' | 'uploading' | 'completed' | 'failed';
  status_detail: string;
  external_id: string;
  error: string;
  created_at: string;
  updated_at: string;
};

function formatTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

export default function VideoLibrary() {
  const [videos, setVideos] = useState<string[]>([]);
  const [publishHistory, setPublishHistory] = useState<DistributionJob[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [videosRes, historyRes] = await Promise.all([
          fetch('/api/videos'),
          fetch('/api/publish/history')
        ]);

        if (videosRes.ok) {
          const data = await videosRes.json();
          setVideos(data || []);
        }

        if (historyRes.ok) {
          const data = await historyRes.json();
          setPublishHistory(data || []);
        }
      } catch (err) {
        console.error('Failed to fetch data:', err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  return (
    <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-20">
      <div className="flex flex-col md:flex-row justify-between items-end gap-6 border-l-4 border-blue-500 pl-6">
        <div>
          <h2 className="text-4xl font-black text-white tracking-tighter uppercase">VIDEO ARCHIVE</h2>
          <p className="text-zinc-500 mt-2 font-medium">Manage and review raw production assets in cold storage.</p>
        </div>
        <div className="bg-zinc-900 border border-white/5 rounded-2xl px-6 py-3 text-xs font-bold text-zinc-500">
            TOTAL ASSETS: {videos.length}
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 shadow-[0_0_20px_rgba(59,130,246,0.3)]"></div>
        </div>
      ) : (
        <div className="grid gap-12 lg:grid-cols-[1fr_1fr]">
          {/* Raw Assets Column */}
          <div className="space-y-6">
            <div className="flex items-center gap-3 mb-2">
               <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center text-lg shadow-lg border border-blue-500/30 text-blue-400">📂</div>
               <h3 className="text-xl font-bold text-white uppercase tracking-tight">Raw Assets</h3>
            </div>
            <div className="grid gap-4 max-h-[800px] overflow-auto pr-2 custom-scrollbar">
              {videos.length > 0 ? (
                videos.map((video) => (
                  <div key={video} className="bg-card border border-white/5 rounded-2xl p-6 flex items-center justify-between hover:border-blue-500/30 transition-all duration-300 group shadow-xl">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 bg-zinc-900 rounded-xl flex items-center justify-center text-2xl shadow-inner border border-white/5 group-hover:scale-110 transition-transform duration-500">
                        🎬
                      </div>
                      <div className="max-w-[200px] md:max-w-xs">
                        <p className="font-mono text-xs font-bold text-white mb-1 group-hover:text-blue-400 transition-colors uppercase tracking-tighter truncate">{video}</p>
                        <span className="text-[9px] font-black text-zinc-600 bg-black/40 px-2 py-0.5 rounded border border-white/5 uppercase tracking-widest">MP4 CONTAINER</span>
                      </div>
                    </div>
                    <div className="flex gap-2">
                       <button className="p-3 bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-white rounded-lg text-[10px] font-black transition-all border border-white/5 active:scale-95 uppercase tracking-widest">
                        DL
                      </button>
                      <button className="px-4 py-3 bg-blue-600/10 text-blue-400 border border-blue-500/20 hover:bg-blue-600 hover:text-white rounded-lg text-[10px] font-black transition-all active:scale-95 uppercase tracking-widest">
                        Preview
                      </button>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-20 bg-card rounded-3xl border-2 border-dashed border-white/5">
                  <p className="text-zinc-600 font-bold uppercase tracking-widest text-xs">No raw assets</p>
                </div>
              )}
            </div>
          </div>

          {/* Publish History Column */}
          <div className="space-y-6">
            <div className="flex items-center gap-3 mb-2">
               <div className="w-8 h-8 bg-emerald-500/20 rounded-lg flex items-center justify-center text-lg shadow-lg border border-emerald-500/30 text-emerald-400">🌐</div>
               <h3 className="text-xl font-bold text-white uppercase tracking-tight">Publish History</h3>
            </div>
            <div className="grid gap-4 max-h-[800px] overflow-auto pr-2 custom-scrollbar">
              {publishHistory.length > 0 ? (
                publishHistory.map((dist) => (
                  <div key={dist.id} className="bg-card border border-white/5 rounded-2xl p-6 hover:border-emerald-500/30 transition-all duration-300 group shadow-xl">
                    <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center gap-3">
                            <div className="w-10 h-10 bg-black/40 rounded-xl flex items-center justify-center border border-white/5 group-hover:border-emerald-500/30 transition-colors">
                                <span className="text-xs font-black text-zinc-400 uppercase">{dist.platform.slice(0, 2)}</span>
                            </div>
                            <div>
                                <p className="text-sm font-black text-white uppercase tracking-tight">{dist.platform}</p>
                                <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">Account: {dist.account_id}</p>
                            </div>
                        </div>
                        <span className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest ${
                            dist.status === 'completed' ? 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30' : 
                            dist.status === 'failed' ? 'bg-red-500/15 text-red-300 border-red-500/30' : 
                            'bg-amber-500/15 text-amber-300 border-amber-500/30'
                        }`}>
                            {dist.status}
                        </span>
                    </div>

                    <div className="grid grid-cols-2 gap-4 bg-black/40 rounded-xl p-4 border border-white/5">
                        <div className="space-y-1">
                            <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">External ID</p>
                            <p className="text-[10px] text-zinc-300 font-mono break-all">{dist.external_id || 'N/A'}</p>
                        </div>
                        <div className="space-y-1">
                            <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">Job ID</p>
                            <p className="text-[10px] text-zinc-300 font-mono break-all">{dist.generation_job_id}</p>
                        </div>
                        <div className="space-y-1 col-span-2">
                            <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">Last Update</p>
                            <p className="text-[10px] text-zinc-400 font-bold uppercase">{formatTime(dist.updated_at)}</p>
                        </div>
                        {dist.error && (
                            <div className="col-span-2 mt-2 bg-red-500/5 border border-red-500/10 rounded-lg p-2">
                                <p className="text-[9px] text-red-400 font-bold uppercase tracking-widest mb-1">Error Report</p>
                                <p className="text-[10px] text-red-300/80 font-medium italic">{dist.error}</p>
                            </div>
                        )}
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-20 bg-card rounded-3xl border-2 border-dashed border-white/5">
                  <p className="text-zinc-600 font-bold uppercase tracking-widest text-xs">No distribution history</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
