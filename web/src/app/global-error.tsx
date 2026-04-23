'use client';

import Link from 'next/link';

const fallbackLinks = [
  { href: '/', label: 'Creator Portal' },
  { href: '/research', label: 'Trend Research Lab' },
  { href: '/clipper', label: 'Clipper Portal' },
  { href: '/videos', label: 'Asset Library' },
  { href: '/integrations', label: 'Integrations' },
];

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-background text-foreground">
        <div className="flex min-h-screen items-center justify-center px-6 py-16">
          <div className="relative w-full max-w-4xl overflow-hidden rounded-[2.5rem] border border-white/5 bg-card p-10 shadow-2xl">
            <div className="absolute -top-24 -right-24 w-80 h-80 rounded-full bg-red-500/5 blur-3xl" />
            <div className="absolute -bottom-24 -left-24 w-80 h-80 rounded-full bg-emerald-500/5 blur-3xl" />

            <div className="relative z-10 space-y-8">
              <div className="border-l-4 border-red-500 pl-6">
                <p className="mb-3 text-[10px] font-black uppercase tracking-[0.2em] text-zinc-500">
                  System Error
                </p>
                <h1 className="text-4xl font-black uppercase tracking-tighter text-white">
                  Operations Shell Halted
                </h1>
                <p className="mt-3 max-w-2xl leading-relaxed text-zinc-400">
                  A runtime error stopped the dashboard shell from rendering. Retry the page, or jump back to a working module below.
                </p>
              </div>

              <div className="rounded-2xl border border-red-500/20 bg-red-500/10 px-5 py-4">
                <p className="text-[10px] font-black uppercase tracking-widest text-red-300">
                  Error Detail
                </p>
                <p className="mt-2 break-words text-sm text-red-100">
                  {error.message || 'Unknown runtime error'}
                </p>
                {error.digest && (
                  <p className="mt-2 text-[10px] font-bold uppercase tracking-widest text-red-200/80">
                    Digest: {error.digest}
                  </p>
                )}
              </div>

              <div className="flex flex-col gap-3 sm:flex-row">
                <button
                  onClick={() => reset()}
                  className="rounded-2xl bg-red-600 px-5 py-3 text-xs font-black uppercase tracking-[0.2em] text-white transition-all hover:bg-red-500 active:scale-95"
                >
                  Retry Shell
                </button>
                <Link
                  href="/"
                  className="rounded-2xl border border-white/10 bg-black/30 px-5 py-3 text-xs font-black uppercase tracking-[0.2em] text-zinc-200 transition-all hover:border-emerald-500/30 hover:text-emerald-300"
                >
                  Back to Creator Portal
                </Link>
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                {fallbackLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    className="group rounded-2xl border border-white/5 bg-black/30 px-5 py-4 transition-all hover:border-white/15 hover:bg-black/50"
                  >
                    <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">
                      Navigate
                    </p>
                    <p className="mt-1 text-sm font-bold text-white group-hover:text-emerald-400 transition-colors">
                      {link.label}
                    </p>
                  </Link>
                ))}
              </div>
            </div>
          </div>
        </div>
      </body>
    </html>
  );
}
