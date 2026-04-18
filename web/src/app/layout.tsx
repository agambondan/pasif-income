'use client';

import { useEffect, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import Link from 'next/link';
import "./globals.css";

type SessionUser = {
  username?: string;
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<SessionUser | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      const savedUser = localStorage.getItem('user');
      if (savedUser) {
        setUser(JSON.parse(savedUser) as SessionUser);
      } else if (pathname !== '/login') {
        router.push('/login');
      }
      setIsReady(true);
    }, 0);

    return () => window.clearTimeout(timer);
  }, [pathname, router]);

  const handleLogout = async () => {
    try {
      await fetch('/api/auth/logout', {
        method: 'POST',
        credentials: 'include',
      });
    } catch (err) {
      console.error('Failed to logout:', err);
    } finally {
      localStorage.removeItem('user');
      router.push('/login');
    }
  };

  if (!isReady) return null;

  if (pathname === '/login') {
    return (
      <html lang="en" className="h-full antialiased">
        <body className="min-h-full bg-gray-950">{children}</body>
      </html>
    );
  }

  return (
    <html lang="en" className="dark">
      <body className="min-h-full flex bg-background text-foreground">
        {/* Left Sidebar */}
        <aside className="w-64 border-r border-white/5 flex flex-col fixed inset-y-0 bg-card/50 backdrop-blur-xl z-30">
          <div className="p-8">
            <h1 className="text-2xl font-black bg-clip-text text-transparent bg-gradient-to-br from-emerald-400 to-blue-500 tracking-tighter">
              CLIPS FACTORY
            </h1>
          </div>
          
          <nav className="flex-1 px-4 space-y-1 mt-2">
            <Link href="/" className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group ${pathname === '/' ? 'bg-emerald-500/10 text-emerald-400 font-bold' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${pathname === '/' ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.6)]' : 'bg-transparent'}`}></span>
              Dashboard
            </Link>
            <Link href="/videos" className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group ${pathname === '/videos' ? 'bg-emerald-500/10 text-emerald-400 font-bold' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${pathname === '/videos' ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.6)]' : 'bg-transparent'}`}></span>
              Video Library
            </Link>
            <Link href="/integrations" className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group ${pathname === '/integrations' ? 'bg-emerald-500/10 text-emerald-400 font-bold' : 'text-zinc-400 hover:text-white hover:bg-white/5'}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${pathname === '/integrations' ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.6)]' : 'bg-transparent'}`}></span>
              Integrations
            </Link>
          </nav>

          <div className="p-6 border-t border-white/5 bg-black/20">
            <div className="flex items-center gap-3 px-2 py-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-emerald-500 to-blue-600 flex items-center justify-center text-white font-black shadow-lg">
                {user?.username?.[0]?.toUpperCase() || 'A'}
              </div>
              <div className="overflow-hidden">
                <p className="text-sm font-bold text-white truncate">{user?.username || 'Admin'}</p>
                <p className="text-[10px] text-emerald-500 font-medium uppercase tracking-widest">Operator</p>
              </div>
            </div>
            <button 
              onClick={handleLogout}
              className="w-full flex items-center justify-center gap-2 px-4 py-2.5 text-xs font-bold text-zinc-400 hover:text-red-400 bg-zinc-800/50 hover:bg-red-500/10 rounded-xl transition-all border border-white/5"
            >
              Sign Out
            </button>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 ml-64 flex flex-col min-h-screen bg-background">
          {/* Section Header */}
          <header className="h-20 border-b border-white/5 flex items-center justify-between px-10 bg-background/80 backdrop-blur-md sticky top-0 z-20">
            <div className="flex items-center gap-4">
               <h2 className="text-lg font-bold text-white uppercase tracking-widest">
                {pathname === '/' ? 'Dashboard' : pathname.replace('/', '').replace('-', ' ')}
               </h2>
            </div>
            <div className="flex items-center gap-6">
               <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-500/10 border border-emerald-500/20 rounded-full">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                  <span className="text-[10px] font-bold text-emerald-500 uppercase tracking-tighter">Live Monitor</span>
               </div>
              <span className="text-[10px] font-mono bg-zinc-800 px-3 py-1 rounded-lg text-zinc-400 border border-white/5">BUILD v1.2.0</span>
            </div>
          </header>

          {/* Section Content */}
          <section className="flex-1 p-10 max-w-7xl w-full mx-auto">
            {children}
          </section>

          {/* Section Footer */}
          <footer className="px-10 py-8 border-t border-white/5 bg-card/30">
            <div className="flex flex-col md:flex-row justify-between items-center gap-4 text-[11px] font-medium text-zinc-500 uppercase tracking-widest">
              <p>© 2026 CLIPS FACTORY // AUTOMATED PIPELINE</p>
              <div className="flex gap-8">
                <a href="#" className="hover:text-emerald-400 transition-colors">Documentation</a>
                <a href="#" className="hover:text-emerald-400 transition-colors">API Status</a>
                <a href="#" className="hover:text-emerald-400 transition-colors">System Logs</a>
              </div>
            </div>
          </footer>
        </main>
      </body>
    </html>
  );
}
