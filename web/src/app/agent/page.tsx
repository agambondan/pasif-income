"use client";

import { useEffect, useState, useRef } from "react";
import { DashboardSurfaceHeader, DashboardEmptyState } from "@/components/dashboard-surface";

type AgentJob = {
  id: string;
  niche: string;
  topic: string;
  current_stage?: string | null;
  progress_pct?: number | null;
  status?: string | null;
};

type AgentEvent = {
  id: string;
  type: "thought" | "action" | "result" | "system";
  content: string;
  timestamp: string;
  metadata?: Record<string, unknown>;
};

const typeStyles: Record<AgentEvent["type"], string> = {
  thought: "border-indigo-500/30 bg-indigo-500/5 text-indigo-300",
  action: "border-amber-500/30 bg-amber-500/5 text-amber-300 font-mono",
  result: "border-emerald-500/30 bg-emerald-500/5 text-emerald-300",
  system: "border-zinc-500/30 bg-zinc-500/5 text-zinc-400 italic",
};

const typeIcons: Record<AgentEvent["type"], string> = {
  thought: "🧠",
  action: "🛠️",
  result: "✅",
  system: "📟",
};

export default function AgentConsole() {
  const [jobs, setJobs] = useState<AgentJob[]>([]);
  const [selectedJobId, setSelectedJobId] = useState<string>("");
  const [eventsByJob, setEventsByJob] = useState<Record<string, AgentEvent[]>>({});
  const [isLive, setIsLive] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Fetch recent jobs
  useEffect(() => {
    const fetchJobs = async () => {
      try {
        const res = await fetch("/api/jobs", { credentials: "include" });
        if (res.ok) {
          const data = await res.json();
          setJobs(data || []);
          if (data && data.length > 0) {
            setSelectedJobId((current) => current || data[0].id);
          }
        }
      } catch (err) {
        console.error("Failed to fetch jobs:", err);
      }
    };
    void fetchJobs();
  }, []);

  // Handle SSE
  useEffect(() => {
    if (!selectedJobId || !isLive) return;

    const eventSource = new EventSource(`/api/agent/events?job_id=${selectedJobId}`);

    eventSource.onmessage = (event) => {
      try {
        const newEvent = JSON.parse(event.data) as AgentEvent;
        setEventsByJob((prev) => {
          const current = prev[selectedJobId] ?? [];
          return {
            ...prev,
            [selectedJobId]: [...current, newEvent],
          };
        });
      } catch (err) {
        console.error("Failed to parse event:", err);
      }
    };

    eventSource.onerror = (err) => {
      console.error("SSE error:", err);
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [selectedJobId, isLive]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [selectedJobId, eventsByJob]);

  const selectedJob = jobs.find((job) => job.id === selectedJobId);
  const events = selectedJobId ? eventsByJob[selectedJobId] ?? [] : [];

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-10">
      <DashboardSurfaceHeader
        eyebrow="Intelligence Console"
        title="Gemini Agent Control"
        description="Observe the internal monologue, tool execution, and real-time reasoning of the Gemini agent as it processes production workflows."
        actions={
          <div className="flex items-center gap-4">
             <select 
                value={selectedJobId} 
                onChange={(e) => setSelectedJobId(e.target.value)}
                className="bg-black/40 border border-white/10 rounded-xl px-4 py-2 text-xs font-bold text-white outline-none focus:border-indigo-500 transition-all"
             >
                <option value="">Select Job...</option>
                {jobs.map(j => (
                  <option key={j.id} value={j.id}>{j.niche}: {j.topic} ({j.id})</option>
                ))}
             </select>

             <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full border ${isLive ? 'bg-indigo-500/10 border-indigo-500/20 text-indigo-400' : 'bg-zinc-800 border-white/5 text-zinc-500'}`}>
                <span className={`w-2 h-2 rounded-full ${isLive ? 'bg-indigo-500 animate-pulse' : 'bg-zinc-600'}`}></span>
                <span className="text-[10px] font-bold uppercase tracking-widest">{isLive ? "Live Stream Active" : "Stream Paused"}</span>
             </div>
             <button 
                onClick={() => setIsLive(!isLive)}
                className="bg-zinc-900 hover:bg-zinc-800 px-4 py-1.5 rounded-xl border border-white/5 text-[10px] font-black uppercase tracking-widest transition-all active:scale-95"
             >
                {isLive ? "Pause Log" : "Resume Log"}
             </button>
          </div>
        }
      />

      <div className="grid gap-8 xl:grid-cols-[1fr_350px]">
        {/* Main Event Stream */}
        <div className="bg-black/40 border border-white/5 rounded-[2.5rem] p-8 shadow-2xl flex flex-col h-[700px]">
          <div className="flex items-center justify-between mb-6 px-4">
            <h3 className="text-xs font-black text-zinc-500 uppercase tracking-[0.2em]">Execution Trace</h3>
            <span className="text-[10px] font-mono text-zinc-600">{events.length} events recorded</span>
          </div>

          <div ref={scrollRef} className="flex-1 overflow-y-auto pr-4 space-y-6 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
            {events.length > 0 ? (
              events.map((event) => (
                <div key={event.id} className="group relative">
                   <div className="absolute -left-2 top-0 bottom-0 w-0.5 bg-gradient-to-b from-white/5 via-white/10 to-transparent group-last:to-transparent"></div>
                   
                   <div className={`ml-6 p-5 rounded-2xl border transition-all duration-300 ${typeStyles[event.type]}`}>
                      <div className="flex items-center justify-between mb-3">
                         <div className="flex items-center gap-2">
                            <span className="text-sm">{typeIcons[event.type]}</span>
                            <span className="text-[10px] font-black uppercase tracking-widest opacity-70">{event.type}</span>
                         </div>
                         <span className="text-[9px] font-mono opacity-50">{new Date(event.timestamp).toLocaleTimeString()}</span>
                      </div>
                      <p className="text-sm leading-relaxed whitespace-pre-wrap">{event.content}</p>
                      
                      {event.metadata && (
                        <div className="mt-4 pt-4 border-t border-white/5 grid grid-cols-2 gap-4">
                           {Object.entries(event.metadata).map(([key, val]) => (
                             <div key={key}>
                                <p className="text-[9px] font-black uppercase tracking-widest opacity-40">{key}</p>
                                <p className="text-[10px] font-mono mt-0.5">{JSON.stringify(val)}</p>
                             </div>
                           ))}
                        </div>
                      )}
                   </div>
                </div>
              ))
            ) : (
              <DashboardEmptyState
                icon="📡"
                title="Waiting for activity"
                description="Connect a task to start streaming agent events in real-time."
              />
            )}
          </div>
        </div>

        {/* Sidebar Info */}
        <div className="space-y-6">
           <div className="bg-card border border-white/5 rounded-[2rem] p-6 shadow-xl">
              <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500 mb-4">Agent Status</p>
              <div className="space-y-4">
                 <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-zinc-400">Model</span>
                    <span className="text-xs font-mono text-white">gemini-2.5-pro</span>
                 </div>
                 <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-zinc-400">Context</span>
                    <span className="text-xs font-mono text-white">128k tokens</span>
                 </div>
                 <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-zinc-400">Uptime</span>
                    <span className="text-xs font-mono text-emerald-400">24d 12h 4m</span>
                 </div>
              </div>
           </div>

           <div className="bg-card border border-white/5 rounded-[2rem] p-6 shadow-xl overflow-hidden relative group">
              <div className="absolute top-0 right-0 p-4 opacity-10 grayscale group-hover:opacity-20 transition-opacity">
                <span className="text-6xl">💎</span>
              </div>
              <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500 mb-4">Active Task</p>
              {selectedJob ? (
                <div className="space-y-2">
                  <p className="text-sm font-black text-white uppercase tracking-tight">{selectedJob.niche}</p>
                  <p className="text-[10px] text-indigo-400 font-bold uppercase tracking-widest">{selectedJob.topic}</p>
                  <div className="mt-4 space-y-2">
                      <div className="flex justify-between text-[9px] font-bold text-zinc-500 uppercase tracking-widest">
                        <span>{selectedJob.current_stage || "Queued"}</span>
                        <span>{selectedJob.progress_pct ?? 0}%</span>
                      </div>
                      <div className="w-full h-1 bg-zinc-800 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-indigo-500 transition-all duration-1000"
                          style={{ width: `${selectedJob.progress_pct ?? 0}%` }}
                        ></div>
                      </div>
                  </div>
                </div>
              ) : (
                <p className="text-xs text-zinc-500">No active job selected.</p>
              )}
           </div>

           <div className="bg-indigo-500/10 border border-indigo-500/20 rounded-[2rem] p-6">
              <p className="text-[10px] font-black uppercase tracking-widest text-indigo-400 mb-2">Gemini Tip</p>
              <p className="text-xs text-indigo-200/80 leading-relaxed italic">
                {"I've optimized the clip selection for high-retention hooks. Segment #2 has the best potential for viral loops on Instagram Reels."}
              </p>
           </div>
        </div>
      </div>
    </div>
  );
}
