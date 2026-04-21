'use client';

import { useEffect, useState } from 'react';

type Platform = {
  id: string;
  name: string;
  supported_methods: string[];
  description: string;
};

type ConnectedAccount = {
  id: string;
  platform_id: string;
  display_name: string;
  auth_method: string;
  email: string;
  profile_path: string;
  browser_status?: string;
  expiry: string;
  created_at: string;
};

function browserStatusMeta(status?: string) {
  switch ((status || '').toLowerCase()) {
    case 'ready':
      return { label: 'READY', className: 'text-emerald-300 bg-emerald-500/10 border-emerald-500/20' };
    case 'needs_login':
      return { label: 'NEEDS LOGIN', className: 'text-amber-300 bg-amber-500/10 border-amber-500/20' };
    case 'missing':
      return { label: 'MISSING', className: 'text-rose-300 bg-rose-500/10 border-rose-500/20' };
    case 'provisioned':
      return { label: 'PROVISIONED', className: 'text-sky-300 bg-sky-500/10 border-sky-500/20' };
    case 'unknown':
      return { label: 'UNKNOWN', className: 'text-zinc-300 bg-zinc-500/10 border-zinc-500/20' };
    default:
      return { label: 'UNSET', className: 'text-zinc-500 bg-zinc-500/10 border-zinc-500/20' };
  }
}

export default function Integrations() {
  const [platforms, setPlatforms] = useState<Platform[]>([]);
  const [accounts, setAccounts] = useState<ConnectedAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [connectEmail, setConnectEmail] = useState<Record<string, string>>({});
  const [manualApi, setManualApi] = useState<Record<string, { name: string; key: string }>>({});
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState<Record<string, boolean>>({});

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [platRes, accRes] = await Promise.all([
          fetch('/api/platforms', { credentials: 'include' }),
          fetch('/api/accounts', { credentials: 'include' })
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

  const setBusyKey = (key: string, value: boolean) => {
    setBusy((prev) => ({ ...prev, [key]: value }));
  };

  const handleOAuthConnect = async (platformId: string) => {
    setBusyKey(`${platformId}:oauth`, true);
    try {
      setStatusMessage(null);
      window.location.assign(`/api/auth/${platformId}?method=api`);
    } finally {
      setBusyKey(`${platformId}:oauth`, false);
    }
  };

  const handleManualApiConnect = async (platformId: string) => {
    const data = manualApi[platformId] || { name: '', key: '' };
    if (!data.key.trim()) {
      alert('API Key must be filled.');
      return;
    }

    setBusyKey(`${platformId}:manual`, true);
    try {
      const res = await fetch('/api/accounts/manual', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platform_id: platformId,
          display_name: data.name.trim() || `${platformId.toUpperCase()} Manual`,
          api_key: data.key.trim(),
        }),
        credentials: 'include',
      });

      if (!res.ok) throw new Error(await res.text());

      const newAcc = await res.json();
      setAccounts((prev) => [...prev, newAcc]);
      setManualApi((prev) => ({ ...prev, [platformId]: { name: '', key: '' } }));
      setStatusMessage(`API Key for ${platformId} connected successfully.`);
    } catch (err) {
      console.error('Failed to connect manual API:', err);
      setStatusMessage(err instanceof Error ? err.message : 'Failed to connect manual API');
    } finally {
      setBusyKey(`${platformId}:manual`, false);
    }
  };

  const handleBrowserConnect = async (platformId: string) => {
    const email = (connectEmail[platformId] || '').trim();
    if (!email) {
      alert('Email akun harus diisi dulu untuk Chromium profile.');
      return;
    }
    setBusyKey(`${platformId}:browser`, true);
    try {
      setStatusMessage(`Opening browser login for ${email}...`);
      const res = await fetch(`/api/auth/${platformId}?method=chromium_profile&email=${encodeURIComponent(email)}`, {
        method: 'GET',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }

      // Refresh accounts list to show the new one
      const accRes = await fetch('/api/accounts', { credentials: 'include' });
      if (accRes.ok) {
        const a = await accRes.json();
        setAccounts(a || []);
      }

      setStatusMessage(`Browser login queued for ${email}. Open the host launcher to open Chromium on your desktop.`);
    } catch (err) {
        console.error('Failed to connect browser:', err);
        setStatusMessage(err instanceof Error ? err.message : 'Failed to connect browser');
    } finally {
      setBusyKey(`${platformId}:browser`, false);
    }
  };

  const handleLaunchBrowser = async (accountId: string) => {
    setBusyKey(`${accountId}:launch`, true);
    try {
      setStatusMessage('Queueing browser login session on host launcher...');
      const res = await fetch(`/api/accounts/${accountId}/launch`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
    } catch (err) {
      console.error('Failed to launch browser session:', err);
      setStatusMessage(err instanceof Error ? err.message : 'Failed to launch browser session');
    } finally {
      setBusyKey(`${accountId}:launch`, false);
    }
  };

  const handleRefreshBrowserStatus = async (accountId: string) => {
    setBusyKey(`${accountId}:status`, true);
    try {
      setStatusMessage('Refreshing browser profile status...');
      const res = await fetch(`/api/accounts/${accountId}/status`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const updated = (await res.json()) as ConnectedAccount;
      setAccounts((prev) => prev.map((acc) => (acc.id === accountId ? updated : acc)));
      setStatusMessage(`Status refreshed: ${updated.browser_status || 'unset'}`);
    } catch (err) {
      console.error('Failed to refresh browser status:', err);
      setStatusMessage(err instanceof Error ? err.message : 'Failed to refresh browser status');
    } finally {
      setBusyKey(`${accountId}:status`, false);
    }
  };

  const handleDisconnect = async (accountId: string) => {
    if (!confirm('Are you sure you want to disconnect this account?')) return;
    try {
      const res = await fetch(`/api/accounts/${accountId}`, { method: 'DELETE', credentials: 'include' });
      if (res.ok) {
        setAccounts(accounts.filter(a => a.id !== accountId));
      }
    } catch (err) {
      console.error('Failed to disconnect:', err);
    }
  };

  const browserAccounts = accounts.filter((a) => a.auth_method === 'chromium_profile');

  return (
    <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-20">
      {statusMessage && (
        <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 px-5 py-4 text-sm font-medium text-emerald-300 flex justify-between items-center">
          <span>{statusMessage}</span>
          <button onClick={() => setStatusMessage(null)} className="text-emerald-500 hover:text-emerald-400">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path></svg>
          </button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500 shadow-[0_0_15px_rgba(16,185,129,0.4)]"></div>
        </div>
      ) : (
        <div className="space-y-20">
          {/* Section 1: API KEYS */}
          <section>
            <div className="flex items-center gap-4 mb-8">
              <div className="h-px flex-1 bg-white/5"></div>
              <h3 className="text-xl font-black text-white tracking-widest uppercase flex items-center gap-3">
                <span className="text-blue-500">◆</span> Social Media API Keys
              </h3>
              <div className="h-px flex-1 bg-white/5"></div>
            </div>

            <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
              {platforms.map((platform) => {
                const apiAccs = accounts.filter((a) => a.platform_id === platform.id && a.auth_method === 'api');
                const pManual = manualApi[platform.id] || { name: '', key: '' };

                return (
                  <div key={platform.id} className="bg-card border border-white/5 rounded-[2.5rem] p-8 flex flex-col hover:border-blue-500/30 transition-all duration-500 group relative overflow-hidden shadow-2xl shadow-black/40">
                    <div className="flex items-center gap-4 mb-6">
                       <div className="w-12 h-12 bg-zinc-900 rounded-xl flex items-center justify-center text-2xl border border-white/5 shadow-inner group-hover:scale-110 transition-transform">
                        {platform.id === 'youtube' ? '📺' : platform.id === 'tiktok' ? '🎵' : '📸'}
                      </div>
                      <div>
                        <h4 className="text-xl font-bold text-white group-hover:text-blue-400 transition-colors">{platform.name}</h4>
                        <p className="text-[10px] font-black text-zinc-500 uppercase tracking-widest">Token Auth</p>
                      </div>
                    </div>

                    <div className="space-y-4 mb-8">
                      {apiAccs.length > 0 ? (
                        apiAccs.map((acc) => (
                          <div key={acc.id} className="flex items-center justify-between gap-3 bg-black/40 p-4 rounded-2xl border border-white/5 group/item">
                            <div className="min-w-0">
                              <p className="text-sm font-bold text-white truncate">{acc.display_name}</p>
                              <p className="text-[10px] text-zinc-500 uppercase tracking-widest truncate">{acc.email || 'API KEY'}</p>
                            </div>
                            <button
                              onClick={() => handleDisconnect(acc.id)}
                              className="text-zinc-600 hover:text-red-400 transition-colors p-1 opacity-0 group-hover/item:opacity-100"
                            >
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                            </button>
                          </div>
                        ))
                      ) : (
                        <div className="py-8 flex flex-col items-center justify-center border border-dashed border-white/5 rounded-2xl bg-black/10">
                          <p className="text-[10px] font-black text-zinc-600 uppercase tracking-widest">No API Accounts</p>
                        </div>
                      )}
                    </div>

                    <div className="mt-auto space-y-4">
                      <div className="p-5 rounded-3xl bg-black/20 border border-white/5 space-y-3">
                         <input
                          value={pManual.name}
                          onChange={(e) => setManualApi(prev => ({ ...prev, [platform.id]: { ...pManual, name: e.target.value } }))}
                          placeholder="Account Name (Optional)"
                          className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-2.5 text-xs text-white outline-none focus:border-blue-500/40 transition-colors"
                        />
                        <input
                          value={pManual.key}
                          onChange={(e) => setManualApi(prev => ({ ...prev, [platform.id]: { ...pManual, key: e.target.value } }))}
                          placeholder="Enter API Key / Token"
                          className="w-full bg-black/40 border border-white/5 rounded-xl px-4 py-2.5 text-xs text-white outline-none focus:border-blue-500/40 transition-colors"
                        />
                        <button
                          onClick={() => handleManualApiConnect(platform.id)}
                          disabled={busy[`${platform.id}:manual`]}
                          className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-[11px] font-black uppercase tracking-widest rounded-xl transition-all shadow-lg shadow-blue-900/20"
                        >
                          {busy[`${platform.id}:manual`] ? 'CONNECTING...' : 'Add API Key'}
                        </button>
                      </div>

                      {platform.id === 'youtube' && (
                        <button
                          onClick={() => handleOAuthConnect(platform.id)}
                          disabled={busy[`${platform.id}:oauth`]}
                          className="w-full py-3 border border-white/10 hover:bg-white/5 text-white text-[11px] font-black uppercase tracking-widest rounded-xl transition-all"
                        >
                           {busy[`${platform.id}:oauth`] ? 'CONNECTING...' : 'Connect with OAuth'}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </section>

          {/* Section 2: BROWSER PROFILES */}
          <section>
            <div className="flex items-center gap-4 mb-8">
              <div className="h-px flex-1 bg-white/5"></div>
              <h3 className="text-xl font-black text-white tracking-widest uppercase flex items-center gap-3">
                <span className="text-emerald-500">◆</span> Browser Profiles
              </h3>
              <div className="h-px flex-1 bg-white/5"></div>
            </div>

            <div className="grid gap-8 lg:grid-cols-3">
              {/* Creator Card */}
              <div className="bg-zinc-900/50 border border-emerald-500/20 rounded-[2.5rem] p-8 flex flex-col shadow-2xl">
                <div className="flex items-center gap-4 mb-6">
                  <div className="w-12 h-12 bg-emerald-500/10 rounded-xl flex items-center justify-center text-2xl border border-emerald-500/20 text-emerald-500 shadow-inner">
                    🌐
                  </div>
                  <div>
                    <h4 className="text-xl font-bold text-white">Create Profile</h4>
                    <p className="text-[10px] font-black text-emerald-500/60 uppercase tracking-widest">New Browser Session</p>
                  </div>
                </div>

                <p className="text-xs text-zinc-500 mb-8 leading-relaxed">
                  Generate a dedicated Chromium profile for a specific account. This allows you to stay logged in and bypass OAuth restrictions during automated publishing.
                </p>

                <div className="space-y-4">
                  <div>
                    <label className="text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1">Select Platform</label>
                    <div className="grid grid-cols-3 gap-2 mt-2">
                      {platforms.map(p => (
                        <button
                          key={p.id}
                          onClick={() => setConnectEmail(prev => ({ ...prev, selectedPlatform: p.id }))}
                          className={`py-2 rounded-xl text-[10px] font-black uppercase tracking-tighter border transition-all ${
                            (connectEmail['selectedPlatform'] || 'youtube') === p.id
                            ? 'bg-emerald-500 border-emerald-500 text-black'
                            : 'bg-black/40 border-white/5 text-zinc-500 hover:border-emerald-500/40'
                          }`}
                        >
                          {p.name}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div>
                    <label className="text-[10px] font-black text-zinc-500 uppercase tracking-widest ml-1">Email Address</label>
                    <input
                      value={connectEmail[connectEmail['selectedPlatform'] || 'youtube'] || ''}
                      onChange={(e) => setConnectEmail(prev => ({ ...prev, [connectEmail['selectedPlatform'] || 'youtube']: e.target.value }))}
                      placeholder="account@gmail.com"
                      className="mt-2 w-full bg-black/40 border border-white/5 rounded-xl px-4 py-3 text-sm text-white outline-none focus:border-emerald-500/40 transition-colors"
                    />
                  </div>

                  <button
                    onClick={() => handleBrowserConnect(connectEmail['selectedPlatform'] || 'youtube')}
                    disabled={busy[`${connectEmail['selectedPlatform'] || 'youtube'}:browser`]}
                    className="w-full py-4 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-[11px] font-black uppercase tracking-widest rounded-2xl transition-all shadow-lg shadow-emerald-900/40"
                  >
                    {busy[`${connectEmail['selectedPlatform'] || 'youtube'}:browser`] ? 'CREATING...' : 'CREATE & OPEN PROFILE'}
                  </button>
                </div>
              </div>

              {/* Profiles List */}
              <div className="lg:col-span-2 grid gap-6 md:grid-cols-2">
                {browserAccounts.length > 0 ? (
                  browserAccounts.map((acc) => (
                    <div key={acc.id} className="bg-black/30 border border-white/5 rounded-[2rem] p-6 hover:border-emerald-500/20 transition-all group relative overflow-hidden">
                      <div className="flex items-start justify-between mb-4">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 bg-zinc-900 rounded-lg flex items-center justify-center text-xl border border-white/5">
                            {acc.platform_id === 'youtube' ? '📺' : acc.platform_id === 'tiktok' ? '🎵' : '📸'}
                          </div>
                          <div>
                            <p className="text-sm font-bold text-white truncate">{acc.display_name}</p>
                            <p className="text-[9px] text-zinc-500 uppercase tracking-widest">{acc.email}</p>
                          </div>
                        </div>
                        <span className={`px-2 py-0.5 rounded-full border text-[8px] font-black uppercase tracking-widest ${browserStatusMeta(acc.browser_status).className}`}>
                          {browserStatusMeta(acc.browser_status).label}
                        </span>
                      </div>

                      <div className="bg-black/40 rounded-xl p-3 mb-4">
                        <p className="text-[8px] font-black text-zinc-600 uppercase tracking-widest mb-1">Local Path</p>
                        <p className="text-[9px] text-zinc-500 truncate font-mono">{acc.profile_path}</p>
                      </div>

                      <div className="grid grid-cols-2 gap-2">
                        <button
                          onClick={() => handleLaunchBrowser(acc.id)}
                          disabled={busy[`${acc.id}:launch`]}
                          className="py-2.5 rounded-xl bg-emerald-600/10 hover:bg-emerald-600/20 text-emerald-500 text-[10px] font-black uppercase tracking-widest transition-all"
                        >
                          {busy[`${acc.id}:launch`] ? 'Opening...' : 'Open Browser'}
                        </button>
                        <button
                          onClick={() => handleRefreshBrowserStatus(acc.id)}
                          disabled={busy[`${acc.id}:status`]}
                          className="py-2.5 rounded-xl border border-white/5 bg-black/40 hover:bg-black/60 text-white text-[10px] font-black uppercase tracking-widest transition-all"
                        >
                          {busy[`${acc.id}:status`] ? 'Wait...' : 'Sync Status'}
                        </button>
                      </div>

                      <button
                        onClick={() => handleDisconnect(acc.id)}
                        className="absolute top-4 right-4 text-zinc-700 hover:text-red-500 transition-colors opacity-0 group-hover:opacity-100"
                        title="Delete Profile"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                      </button>
                    </div>
                  ))
                ) : (
                  <div className="col-span-full h-full flex flex-col items-center justify-center border border-dashed border-white/5 rounded-[2.5rem] bg-black/10 py-20">
                    <div className="text-4xl mb-4 opacity-20 text-emerald-500">🪹</div>
                    <p className="text-[11px] font-black text-zinc-600 uppercase tracking-widest">No profiles configured yet</p>
                  </div>
                )}
              </div>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}
