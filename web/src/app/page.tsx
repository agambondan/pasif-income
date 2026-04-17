'use client';

import { useEffect, useState } from 'react';
import { Clip } from '@/types/clip';

export default function Dashboard() {
  const [clips, setClips] = useState<Clip[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchClips = async () => {
    try {
      const res = await fetch('http://localhost:8080/api/clips');
      const data = await res.json();
      setClips(data);
    } catch (err) {
      console.error('Failed to fetch clips:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchClips();
  }, []);

  const updateStatus = async (id: string, status: Clip['status']) => {
    try {
      await fetch('http://localhost:8080/api/clips', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, status }),
      });
      fetchClips(); // Refresh list
    } catch (err) {
      console.error('Failed to update status:', err);
    }
  };

  return (
    <main className="min-h-screen bg-gray-900 text-white p-8">
      <header className="mb-12 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-4xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-emerald-400">
            Podcast Clips Factory
          </h1>
          <p className="text-gray-400 mt-2">AI-Generated Viral Content Review Dashboard</p>
        </div>
        <div className="flex gap-4">
            <button 
                onClick={fetchClips}
                className="bg-gray-800 hover:bg-gray-700 px-4 py-2 rounded-lg border border-gray-700 text-sm transition-colors"
            >
                Refresh Queue
            </button>
            <div className="bg-emerald-500/10 px-4 py-2 rounded-lg border border-emerald-500/20">
                <span className="text-emerald-400 text-sm font-bold">● Backend Online</span>
            </div>
        </div>
      </header>

      {loading ? (
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-8">
          {clips && clips.length > 0 ? (
            clips.map((clip) => (
              <div key={clip.id} className="bg-gray-800 rounded-2xl overflow-hidden border border-gray-700 shadow-2xl flex flex-col transition-all hover:border-gray-500">
                {/* Real Video Player */}
                <div className="aspect-[9/16] bg-black relative group">
                  <video 
                    src={clip.s3_path} 
                    controls 
                    className="w-full h-full object-contain"
                    poster="/video-poster.png" // Fallback poster
                  />
                  
                  {/* Floating Viral Score */}
                  <div className="absolute top-4 right-4 z-10 bg-black/60 backdrop-blur-md text-emerald-400 px-3 py-1.5 rounded-full text-xs font-black border border-emerald-500/30">
                    🔥 {clip.viral_score}% VIRAL
                  </div>
                </div>

                <div className="p-5 flex-1 flex flex-col">
                  <h3 className="text-lg font-bold mb-1 line-clamp-2 leading-tight h-12">{clip.headline}</h3>
                  <p className="text-xs text-gray-500 mb-4 font-mono uppercase tracking-widest">
                    TS: {clip.start_time} - {clip.end_time}
                  </p>
                  
                  <div className="mt-auto space-y-3">
                    <div className="flex items-center justify-between text-xs px-1">
                        <span className="text-gray-400 italic">Current Status:</span>
                        <span className={`font-bold px-2 py-0.5 rounded ${
                            clip.status === 'approved' ? 'bg-emerald-500/20 text-emerald-400' : 
                            clip.status === 'rejected' ? 'bg-red-500/20 text-red-400' : 'bg-yellow-500/20 text-yellow-400'
                        }`}>
                            {clip.status.toUpperCase()}
                        </span>
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                        <button 
                        onClick={() => updateStatus(clip.id, 'approved')}
                        disabled={clip.status === 'approved'}
                        className={`font-bold py-2.5 rounded-xl transition-all ${
                            clip.status === 'approved' 
                            ? 'bg-emerald-900/20 text-emerald-700 cursor-not-allowed' 
                            : 'bg-emerald-600 hover:bg-emerald-500 text-white shadow-lg shadow-emerald-900/20'
                        }`}
                        >
                        Approve
                        </button>
                        <button 
                        onClick={() => updateStatus(clip.id, 'rejected')}
                        disabled={clip.status === 'rejected'}
                        className={`font-bold py-2.5 rounded-xl transition-all border ${
                            clip.status === 'rejected'
                            ? 'bg-red-900/10 text-red-900 border-red-900/20 cursor-not-allowed'
                            : 'bg-gray-700 hover:bg-red-900/40 hover:text-red-400 text-white border-gray-600'
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
            <div className="col-span-full text-center py-32 bg-gray-800/30 rounded-3xl border-2 border-dashed border-gray-800">
              <div className="text-6xl mb-4">🎬</div>
              <p className="text-gray-500 text-xl font-medium">No clips found in the queue.</p>
              <p className="text-gray-600 mt-1">Start the Go pipeline to generate viral content.</p>
            </div>
          )}
        </div>
      )}
    </main>
  );
}
