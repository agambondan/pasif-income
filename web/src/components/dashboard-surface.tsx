'use client';

import type { ReactNode } from 'react';

type DashboardSurfaceHeaderProps = {
  eyebrow: string;
  title: string;
  description: string;
  actions?: ReactNode;
};

export function DashboardSurfaceHeader({
  eyebrow,
  title,
  description,
  actions,
}: DashboardSurfaceHeaderProps) {
  return (
    <div className="flex flex-col gap-6 border-l-4 border-emerald-500/80 pl-6">
      <div className="space-y-2">
        <p className="text-[10px] font-black uppercase tracking-[0.24em] text-zinc-500">
          {eyebrow}
        </p>
        <h2 className="text-4xl font-black tracking-tighter text-white uppercase">
          {title}
        </h2>
        <p className="max-w-3xl text-sm font-medium leading-relaxed text-zinc-500">
          {description}
        </p>
      </div>

      {actions && <div className="flex flex-wrap gap-3">{actions}</div>}
    </div>
  );
}

type DashboardEmptyStateProps = {
  icon: string;
  title: string;
  description: string;
  action?: ReactNode;
  compact?: boolean;
};

export function DashboardEmptyState({
  icon,
  title,
  description,
  action,
  compact = false,
}: DashboardEmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center rounded-[2.5rem] border border-dashed border-white/5 bg-black/10 text-center ${compact ? 'py-12' : 'py-20'}`}
    >
      <div className="text-5xl opacity-20 grayscale">{icon}</div>
      <p className="mt-4 text-[11px] font-black uppercase tracking-[0.24em] text-zinc-500">
        {title}
      </p>
      <p className="mt-3 max-w-xl text-sm text-zinc-500">{description}</p>
      {action && <div className="mt-6">{action}</div>}
    </div>
  );
}
