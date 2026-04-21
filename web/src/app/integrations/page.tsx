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

  const handleApiConnect = async (platformId: string) => {
    setBusyKey(`${platformId}:api`, true);
    try {
      setStatusMessage(null);
      window.location.assign(`/api/auth/${platformId}?method=api`);
    } finally {
      setBusyKey(`${platformId}:api`, false);
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
      setStatusMessage(`Browser login queued for ${email}. Open the host launcher to open Chromium on your desktop.`);
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

  return (
    <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700 pt-10">
      <div className="border-l-4 border-emerald-500 pl-6">
        <h2 className="text-4xl font-black text-white tracking-tighter">INTEGRATIONS</h2>
        <p className="text-zinc-500 mt-2 font-medium">
          Separate API connections and Chromium profiles. Profile connects open a login browser once, then reuse the saved profile on publish.
        </p>
        <div className="mt-4 space-y-2 rounded-2xl border border-white/5 bg-black/20 px-4 py-3 max-w-3xl">
          <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
            API Auth is for token-based publishing and analytics.
          </p>
          <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
            Chromium Profile is for email lookup, one-time login setup, and profile reuse during publish.
          </p>
          <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
            Use Open Login after Create & Open Profile if the browser session needs manual sign-in.
          </p>
        </div>
      </div>

      {statusMessage && (
        <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 px-5 py-4 text-sm font-medium text-emerald-300">
          {statusMessage}
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-500 shadow-[0_0_15px_rgba(16,185,129,0.4)]"></div>
        </div>
      ) : (
        <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {platforms.map((platform) => {
            const apiAccounts = accounts.filter((a) => a.platform_id === platform.id && a.auth_method === 'api');
            const browserAccounts = accounts.filter((a) => a.platform_id === platform.id && a.auth_method === 'chromium_profile');
            const supportsApi = platform.supported_methods.includes('api');
            const supportsBrowser = platform.supported_methods.includes('chromium_profile');
            
            return (
              <div key={platform.id} className="bg-card border border-white/5 rounded-[2rem] p-8 flex flex-col hover:border-emerald-500/30 transition-all duration-300 group relative overflow-hidden shadow-2xl shadow-black/40">
                <div className="absolute -top-24 -right-24 w-48 h-48 bg-emerald-500/5 rounded-full blur-3xl group-hover:bg-emerald-500/10 transition-colors"></div>
                
                <div className="flex justify-between items-start mb-8 relative z-10">
                  <div className="w-16 h-16 bg-zinc-900 rounded-2xl flex items-center justify-center text-4xl shadow-inner border border-white/5 group-hover:scale-110 transition-transform duration-500 group-hover:shadow-emerald-500/20 group-hover:shadow-lg">
                    {platform.id === 'youtube' ? '📺' : platform.id === 'tiktok' ? '🎵' : '📸'}
                  </div>
                  {(apiAccounts.length > 0 || browserAccounts.length > 0) && (
                    <span className="text-[10px] font-bold text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full animate-pulse">ACTIVE</span>
                  )}
                </div>
                
                <h3 className="text-2xl font-bold text-white mb-2 group-hover:text-emerald-400 transition-colors">{platform.name}</h3>
                <p className="text-zinc-400 text-sm mb-8 leading-relaxed font-medium">{platform.description}</p>

                <div className="grid gap-5 lg:grid-cols-2 relative z-10">
                  <div className="rounded-[1.5rem] border border-white/5 bg-black/30 p-5">
                    <div className="flex items-center justify-between gap-3 mb-4">
                      <div>
                        <p className="text-[10px] font-black text-zinc-500 uppercase tracking-widest">API Auth</p>
                        <h4 className="text-lg font-black text-white">Token Based</h4>
                      </div>
                      <span className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Clean / direct</span>
                    </div>
                    <div className="space-y-3">
                      {supportsApi ? (
                        apiAccounts.length > 0 ? (
                          apiAccounts.map((acc) => (
                            <div key={acc.id} className="flex items-center justify-between gap-3 bg-black/40 p-4 rounded-2xl border border-white/5">
                              <div className="min-w-0">
                                <p className="text-sm font-bold text-white truncate">{acc.display_name}</p>
                                <p className="text-[10px] text-zinc-500 uppercase tracking-widest truncate">{acc.email || 'API account'}</p>
                              </div>
                              <button
                                onClick={() => handleDisconnect(acc.id)}
                                className="text-zinc-600 hover:text-red-400 transition-colors p-1 flex-shrink-0"
                                title="Disconnect"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                              </button>
                            </div>
                          ))
                        ) : (
                          <div className="h-24 flex items-center justify-center border border-dashed border-white/5 rounded-2xl">
                            <p className="text-[11px] font-bold text-zinc-600 uppercase tracking-widest">No API accounts</p>
                          </div>
                        )
                      ) : (
                        <div className="h-24 flex items-center justify-center border border-dashed border-white/5 rounded-2xl">
                          <p className="text-[11px] font-bold text-zinc-600 uppercase tracking-widest">Not supported</p>
                        </div>
                      )}
                      {supportsApi && (
                        <button
                          onClick={() => handleApiConnect(platform.id)}
                          disabled={busy[`${platform.id}:api`]}
                          className="w-full py-4 bg-blue-600 hover:bg-blue-500 text-white font-black rounded-2xl transition-all duration-300 transform active:scale-95 shadow-lg shadow-blue-900/20"
                        >
                          {busy[`${platform.id}:api`] ? 'CONNECTING...' : 'CONNECT API'}
                        </button>
                      )}
                    </div>
                  </div>

                  <div className="rounded-[1.5rem] border border-emerald-500/10 bg-emerald-500/5 p-5">
                    <div className="flex items-center justify-between gap-3 mb-4">
                      <div>
                        <p className="text-[10px] font-black text-zinc-500 uppercase tracking-widest">Browser Profile</p>
                        <h4 className="text-lg font-black text-white">Chrome Login</h4>
                      </div>
                      <span className="text-[9px] font-black uppercase tracking-widest text-emerald-300">Email lookup</span>
                    </div>
                    <p className="mb-4 text-[10px] font-bold text-zinc-500 uppercase tracking-widest leading-relaxed">
                      Connect once, sign in in the opened browser, then reuse the profile on publish without opening Chrome again.
                    </p>

                    <label className="mb-4 block">
                      <span className="text-[10px] font-black text-zinc-500 uppercase tracking-widest">Profile Email</span>
                      <input
                        value={connectEmail[platform.id] || ''}
                        onChange={(event) => setConnectEmail((prev) => ({ ...prev, [platform.id]: event.target.value }))}
                        placeholder="agam.pro234@gmail.com"
                        className="mt-2 w-full rounded-2xl border border-white/10 bg-black/40 px-4 py-3 text-sm text-zinc-100 outline-none transition-colors placeholder:text-zinc-600 focus:border-emerald-500/40"
                      />
                    </label>

                    <div className="space-y-3">
                      {supportsBrowser ? (
                        browserAccounts.length > 0 ? (
                          browserAccounts.map((acc) => (
                            <div key={acc.id} className="bg-black/40 p-4 rounded-2xl border border-emerald-500/10">
                              <div className="flex items-start justify-between gap-3">
                                <div className="min-w-0">
                                  <p className="text-sm font-bold text-white truncate">{acc.display_name}</p>
                                  <p className="text-[10px] text-zinc-500 uppercase tracking-widest truncate">{acc.email}</p>
                                  <p className="text-[10px] text-zinc-600 mt-1 break-all">Profile: {acc.profile_path || '-'}</p>
                                  {acc.auth_method === 'chromium_profile' && (
                                    <span className={`inline-flex items-center mt-2 px-2.5 py-1 rounded-full border text-[9px] font-black uppercase tracking-widest ${browserStatusMeta(acc.browser_status).className}`}>
                                      {browserStatusMeta(acc.browser_status).label}
                                    </span>
                                  )}
                                </div>
                                <div className="flex flex-col gap-2">
                                  <button
                                    onClick={() => handleLaunchBrowser(acc.id)}
                                    disabled={busy[`${acc.id}:launch`]}
                                    className="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-[10px] font-black uppercase tracking-widest transition-all disabled:opacity-50"
                                  >
                                    {busy[`${acc.id}:launch`] ? 'OPENING...' : 'Open Login'}
                                  </button>
                                  <button
                                    onClick={() => handleRefreshBrowserStatus(acc.id)}
                                    disabled={busy[`${acc.id}:status`]}
                                    className="px-4 py-2 rounded-xl border border-white/10 bg-black/30 hover:bg-white/5 text-white text-[10px] font-black uppercase tracking-widest transition-all disabled:opacity-50"
                                  >
                                    {busy[`${acc.id}:status`] ? 'REFRESHING...' : 'Refresh Status'}
                                  </button>
                                  <button
                                    onClick={() => handleDisconnect(acc.id)}
                                    className="text-zinc-600 hover:text-red-400 transition-colors p-1 self-center"
                                    title="Disconnect"
                                  >
                                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                                  </button>
                                </div>
                              </div>
                            </div>
                          ))
                        ) : (
                          <div className="h-24 flex items-center justify-center border border-dashed border-emerald-500/15 rounded-2xl bg-black/20">
                            <p className="text-[11px] font-bold text-zinc-600 uppercase tracking-widest">No browser profiles</p>
                          </div>
                        )
                      ) : (
                        <div className="h-24 flex items-center justify-center border border-dashed border-white/5 rounded-2xl">
                          <p className="text-[11px] font-bold text-zinc-600 uppercase tracking-widest">Not supported</p>
                        </div>
                      )}

                      {supportsBrowser && (
                        <button
                          onClick={() => handleBrowserConnect(platform.id)}
                          disabled={busy[`${platform.id}:browser`]}
                          className="w-full py-4 bg-emerald-600 hover:bg-emerald-500 text-white font-black rounded-2xl transition-all duration-300 transform active:scale-95 shadow-lg shadow-emerald-900/20 relative z-10"
                        >
                          {busy[`${platform.id}:browser`] ? 'OPENING...' : 'CREATE & OPEN PROFILE'}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
