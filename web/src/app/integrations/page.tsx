'use client';

import { useEffect, useState } from 'react';

type Platform = {
  id: string;
  name: string;
  auth_type: string;
  description: string;
};

type ConnectedAccount = {
  id: string;
  platform_id: string;
  display_name: string;
  expiry: string;
  created_at: string;
};

export default function Integrations() {
  const [platforms, setPlatforms] = useState<Platform[]>([]);
  const [accounts, setAccounts] = useState<ConnectedAccount[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [platRes, accRes] = await Promise.all([
          fetch('/api/platforms'),
          fetch('/api/accounts')
        ]);
        
        if (platRes.ok) {
          const p = await platRes.json();
          setPlatforms(p || []);
        }
        if (accRes.ok) {
          const a = await accRes.json();
          setAccounts(a || []);
        }
      } catch (err) {
        console.error('Failed to fetch data:', err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const handleConnect = (platformId: string) => {
    window.location.href = `/api/auth/${platformId}`;
  };

  const handleDisconnect = async (accountId: string) => {
    if (!confirm('Are you sure you want to disconnect this account?')) return;
    try {
      const res = await fetch(`/api/accounts/${accountId}`, { method: 'DELETE' });
      if (res.ok) {
        setAccounts(accounts.filter(a => a.id !== accountId));
      }
    } catch (err) {
      console.error('Failed to disconnect:', err);
    }
  };

  return (
    <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div className="border-l-4 border-emerald-500 pl-6">
        <h2 className="text-4xl font-black text-white tracking-tighter">INTEGRATIONS</h2>
        <p className="text-zinc-500 mt-2 font-medium">Connect and manage your content distribution channels.</p>
      </div>

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500 shadow-[0_0_15px_rgba(16,185,129,0.4)]"></div>
        </div>
      ) : (
        <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {platforms.map((platform) => {
            const connectedForPlatform = accounts.filter(a => a.platform_id === platform.id);
            
            return (
              <div key={platform.id} className="bg-card border border-white/5 rounded-[2rem] p-8 flex flex-col hover:border-emerald-500/30 transition-all duration-300 group relative overflow-hidden shadow-2xl shadow-black/40">
                {/* Background Decor */}
                <div className="absolute -top-24 -right-24 w-48 h-48 bg-emerald-500/5 rounded-full blur-3xl group-hover:bg-emerald-500/10 transition-colors"></div>
                
                <div className="flex justify-between items-start mb-8 relative z-10">
                  <div className="w-16 h-16 bg-zinc-900 rounded-2xl flex items-center justify-center text-4xl shadow-inner border border-white/5 group-hover:scale-110 transition-transform duration-500 group-hover:shadow-emerald-500/20 group-hover:shadow-lg">
                    {platform.id === 'youtube' ? '📺' : platform.id === 'tiktok' ? '🎵' : '📸'}
                  </div>
                  <div className="flex flex-col items-end">
                    <span className="text-[10px] uppercase tracking-[0.2em] font-black px-3 py-1 bg-zinc-900 border border-white/5 rounded-full text-zinc-400">
                      {platform.auth_type}
                    </span>
                    {connectedForPlatform.length > 0 && (
                      <span className="mt-2 text-[10px] font-bold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full animate-pulse">ACTIVE</span>
                    )}
                  </div>
                </div>
                
                <h3 className="text-2xl font-bold text-white mb-2 group-hover:text-emerald-400 transition-colors">{platform.name}</h3>
                <p className="text-zinc-400 text-sm mb-8 leading-relaxed font-medium">{platform.description}</p>

                <div className="flex-1 relative z-10">
                    {connectedForPlatform.length > 0 ? (
                    <div className="mb-8 space-y-3">
                        <p className="text-[10px] font-black text-zinc-500 uppercase tracking-widest mb-2">Connected Accounts</p>
                        {connectedForPlatform.map(acc => (
                        <div key={acc.id} className="flex items-center justify-between bg-black/40 p-4 rounded-2xl border border-white/5 hover:border-emerald-500/20 transition-colors group/acc">
                            <div className="flex items-center gap-3 overflow-hidden">
                            <div className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)] flex-shrink-0"></div>
                            <span className="text-sm font-bold text-zinc-200 truncate" title={acc.display_name}>
                                {acc.display_name}
                            </span>
                            </div>
                            <button 
                            onClick={() => handleDisconnect(acc.id)}
                            className="text-zinc-600 hover:text-red-400 transition-colors p-1"
                            title="Disconnect"
                            >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>
                        ))}
                    </div>
                    ) : (
                        <div className="mb-8 h-20 flex items-center justify-center border-2 border-dashed border-white/5 rounded-2xl">
                             <p className="text-[11px] font-bold text-zinc-600 uppercase tracking-widest">No Connections</p>
                        </div>
                    )}
                </div>
                
                <button 
                  onClick={() => handleConnect(platform.id)}
                  className="w-full py-4 bg-emerald-600 hover:bg-emerald-500 text-white font-black rounded-2xl transition-all duration-300 transform active:scale-95 shadow-lg shadow-emerald-900/20 relative z-10"
                >
                  CONNECT NEW ACCOUNT
                </button>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
