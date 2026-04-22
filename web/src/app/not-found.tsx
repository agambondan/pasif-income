'use client';

import Link from 'next/link';

const quickLinks = [
  { href: '/', label: 'Creator Portal' },
  { href: '/research', label: 'Trend Research Lab' },
  { href: '/clipper', label: 'Clipper Portal' },
  { href: '/videos', label: 'Asset Library' },
  { href: '/agent', label: 'Agent Console' },
  { href: '/integrations', label: 'Integrations' },
];

export default function NotFound() {
  return (
    <div className="flex min-h-[70vh] items-center justify-center py-16">
      <div className="w-full max-w-3xl rounded-[2.5rem] border border-white/5 bg-card p-10 shadow-2xl relative overflow-hidden">
        <div className="absolute -top-20 -right-20 w-72 h-72 bg-emerald-500/5 rounded-full blur-3xl" />
        <div className="absolute -bottom-20 -left-20 w-72 h-72 bg-blue-500/5 rounded-full blur-3xl" />

        <div className="relative z-10 space-y-8">
          <div className="border-l-4 border-emerald-500 pl-6">
            <p className="text-[10px] font-black text-zinc-500 uppercase tracking-[0.2em] mb-3">
              Route Missing
            </p>
            <h1 className="text-4xl font-black text-white tracking-tighter uppercase">
              Page Not Found
            </h1>
            <p className="mt-3 text-zinc-400 max-w-2xl leading-relaxed">
              The requested route does not exist in this operations dashboard. Use the links below to return to a working module.
            </p>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            {quickLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="group rounded-2xl border border-white/5 bg-black/30 px-5 py-4 transition-all hover:border-emerald-500/30 hover:bg-black/50"
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

          <div className="flex flex-col gap-3 rounded-2xl border border-white/5 bg-black/20 px-5 py-4">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">
              Suggested next step
            </p>
            <p className="text-sm text-zinc-300">
              Go back to Creator Portal if you want to generate content, or open Integrations if you need to connect accounts first.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
