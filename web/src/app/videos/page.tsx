'use client';

import { useEffect, useState } from 'react';

export default function VideoLibrary() {
  const [videos, setVideos] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchVideos = async () => {
      try {
        const res = await fetch('/api/videos');
        if (res.ok) {
          const data = await res.json();
          setVideos(data || []);
        }
      } catch (err) {
        console.error('Failed to fetch videos:', err);
      } finally {
        setLoading(false);
      }
    };
    fetchVideos();
  }, []);

  return (
    <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div className="border-l-4 border-blue-500 pl-6">
        <h2 className="text-4xl font-black text-white tracking-tighter uppercase">VIDEO ARCHIVE</h2>
        <p className="text-zinc-500 mt-2 font-medium">Manage and review raw production assets in cold storage.</p>
      </div>

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 shadow-[0_0_20px_rgba(59,130,246,0.3)]"></div>
        </div>
      ) : (
        <div className="grid gap-6">
          {videos.length > 0 ? (
            videos.map((video) => (
              <div key={video} className="bg-card border border-white/5 rounded-3xl p-8 flex items-center justify-between hover:border-blue-500/30 transition-all duration-300 group shadow-2xl">
                <div className="flex items-center gap-6">
                  <div className="w-16 h-16 bg-zinc-900 rounded-2xl flex items-center justify-center text-3xl shadow-inner border border-white/5 group-hover:scale-110 transition-transform duration-500 group-hover:shadow-blue-500/20 group-hover:shadow-lg">
                    🎬
                  </div>
                  <div>
                    <p className="font-mono text-sm font-bold text-white mb-1 group-hover:text-blue-400 transition-colors uppercase tracking-tighter">{video}</p>
                    <div className="flex items-center gap-4">
                        <span className="text-[10px] font-black text-zinc-600 bg-black/40 px-2 py-0.5 rounded border border-white/5 uppercase tracking-widest">MP4 CONTAINER</span>
                        <span className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">MINIO OBJECT STORE</span>
                    </div>
                  </div>
                </div>
                <div className="flex gap-4">
                   <button className="px-6 py-3 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-white rounded-xl text-xs font-black transition-all border border-white/5 active:scale-95 uppercase tracking-widest">
                    Download
                  </button>
                  <button className="px-6 py-3 bg-blue-600/10 text-blue-400 border border-blue-500/20 hover:bg-blue-600 hover:text-white rounded-xl text-xs font-black transition-all active:scale-95 shadow-lg shadow-blue-900/10 uppercase tracking-widest">
                    Launch Preview
                  </button>
                </div>
              </div>
            ))
          ) : (
            <div className="text-center py-40 bg-card rounded-[3rem] border-2 border-dashed border-white/5">
              <div className="text-7xl mb-6 opacity-30 grayscale text-blue-500">📂</div>
              <p className="text-zinc-500 text-xl font-black uppercase tracking-[0.2em]">Storage Empty</p>
              <p className="text-zinc-600 mt-2 font-medium">No raw production assets detected in MinIO.</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
