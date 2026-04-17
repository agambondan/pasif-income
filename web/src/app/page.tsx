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

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

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
  const [lastUpdated, setLastUpdated] = useState('Never');
  const [niche, setNiche] = useState('stoicism');
  const [topic, setTopic] = useState('how to control your mind');

  const fetchClips = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/clips`);
      const data = await res.json();
      setClips(data);
    } catch (err) {
      console.error('Failed to fetch clips:', err);
    }
  }, []);

  const fetchJobs = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/api/jobs`);
      const data = await res.json();
      setJobs(data);
    } catch (err) {
      console.error('Failed to fetch jobs:', err);
    }
  }, []);

  const refreshAll = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchClips(), fetchJobs()]);
    setLoading(false);
    setLastUpdated(new Date().toLocaleString());
  }, [fetchClips, fetchJobs]);

  const pollState = useCallback(async () => {
    await Promise.all([fetchClips(), fetchJobs()]);
    setLastUpdated(new Date().toLocaleString());
  }, [fetchClips, fetchJobs]);

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
      await fetch(`${API_BASE_URL}/api/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ niche, topic }),
      });
      await Promise.all([fetchClips(), fetchJobs()]);
    } catch (err) {
      console.error('Failed to start generation:', err);
    } finally {
      setIsGenerating(false);
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
                onClick={refreshAll}
                className="bg-gray-800 hover:bg-gray-700 px-4 py-2 rounded-lg border border-gray-700 text-sm transition-colors"
            >
                Refresh Queue
            </button>
            <div className="bg-emerald-500/10 px-4 py-2 rounded-lg border border-emerald-500/20">
                <span className="text-emerald-400 text-sm font-bold">● Backend Online</span>
            </div>
            <div className="bg-gray-800/60 px-4 py-2 rounded-lg border border-gray-700 text-sm text-gray-300">
                Last updated: <span className="text-white font-semibold">{lastUpdated}</span>
            </div>
        </div>
      </header>

      <section className="mb-10 grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
        <div className="rounded-3xl border border-gray-800 bg-gray-900/70 p-6 shadow-2xl shadow-black/20">
          <p className="text-xs uppercase tracking-[0.3em] text-blue-400 mb-3">Generate</p>
          <h2 className="text-2xl font-semibold mb-2">Start a new faceless content job</h2>
          <p className="text-gray-400 mb-5">
            Trigger the creator pipeline from the dashboard and watch the job status below.
          </p>
          <div className="grid gap-4 md:grid-cols-2">
            <label className="block">
              <span className="text-sm text-gray-400">Niche</span>
              <input
                value={niche}
                onChange={(e) => setNiche(e.target.value)}
                className="mt-2 w-full rounded-xl border border-gray-700 bg-black/30 px-4 py-3 text-white outline-none transition-colors focus:border-emerald-500"
                placeholder="stoicism"
              />
            </label>
            <label className="block">
              <span className="text-sm text-gray-400">Topic</span>
              <input
                value={topic}
                onChange={(e) => setTopic(e.target.value)}
                className="mt-2 w-full rounded-xl border border-gray-700 bg-black/30 px-4 py-3 text-white outline-none transition-colors focus:border-emerald-500"
                placeholder="how to control your mind"
              />
            </label>
          </div>
          <div className="mt-5 flex flex-wrap gap-3">
            <button
              onClick={startGeneration}
              disabled={isGenerating}
              className="rounded-xl bg-emerald-600 px-5 py-3 font-semibold text-white transition-colors hover:bg-emerald-500 disabled:cursor-not-allowed disabled:bg-emerald-900/40"
            >
              {isGenerating ? 'Starting...' : 'Start Generation'}
            </button>
            <button
              onClick={refreshAll}
              className="rounded-xl border border-gray-700 bg-gray-800 px-5 py-3 font-semibold text-white transition-colors hover:bg-gray-700"
            >
              Refresh State
            </button>
          </div>
        </div>

        <div className="rounded-3xl border border-gray-800 bg-gray-900/70 p-6">
          <p className="text-xs uppercase tracking-[0.3em] text-emerald-400 mb-3">Jobs</p>
          <h2 className="text-2xl font-semibold mb-4">Recent generation jobs</h2>
          <div className="space-y-3">
            {jobs.length > 0 ? jobs.map((job) => (
              <div key={job.id} className="rounded-2xl border border-gray-800 bg-black/20 p-4">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <p className="font-semibold">{job.niche}</p>
                    <p className="text-sm text-gray-400 line-clamp-1">{job.topic}</p>
                  </div>
                  <span className={`rounded-full border px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] ${jobStatusStyles[job.status]}`}>
                    {job.status}
                  </span>
                </div>
                <p className="mt-2 text-xs text-gray-500">
                  Created {formatTime(job.created_at)}
                </p>
                <p className="text-xs text-gray-500">
                  Updated {formatTime(job.updated_at)}
                </p>
                {job.error ? <p className="mt-2 text-sm text-red-300">{job.error}</p> : null}
              </div>
            )) : (
              <div className="rounded-2xl border border-dashed border-gray-800 bg-black/10 p-6 text-sm text-gray-500">
                No generation jobs yet.
              </div>
            )}
          </div>
        </div>
      </section>

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
