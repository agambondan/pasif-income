'use client';

export type ConnectedAccount = {
  id: string;
  platform_id: string;
  display_name: string;
  auth_method?: string;
  browser_status?: string;
};

type AccountSelectorProps = {
  accounts: ConnectedAccount[];
  selectedIds: string[];
  onChange: (nextSelectedIds: string[]) => void;
  emptyMessage: string;
  title: string;
  subtitle: string;
  selectionHint?: string;
};

function formatPlatformName(platformId: string) {
  switch (platformId) {
    case 'youtube':
      return 'YouTube';
    case 'tiktok':
      return 'TikTok';
    case 'instagram':
      return 'Instagram';
    default:
      return platformId.replace(/[-_]/g, ' ');
  }
}

function formatMethodLabel(method?: string) {
  switch ((method || '').toLowerCase()) {
    case 'api':
      return {
        label: 'API',
        className: 'text-sky-300 bg-sky-500/10 border-sky-500/20',
      };
    case 'chromium_profile':
      return {
        label: 'Browser',
        className: 'text-emerald-300 bg-emerald-500/10 border-emerald-500/20',
      };
    default:
      return {
        label: 'UNSET',
        className: 'text-zinc-400 bg-zinc-500/10 border-zinc-500/20',
      };
  }
}

function browserStatusMeta(status?: string) {
  switch ((status || '').toLowerCase()) {
    case 'ready':
      return {
        label: 'READY',
        className: 'text-emerald-300 bg-emerald-500/10 border-emerald-500/20',
      };
    case 'needs_login':
      return {
        label: 'NEEDS LOGIN',
        className: 'text-amber-300 bg-amber-500/10 border-amber-500/20',
      };
    case 'missing':
      return {
        label: 'MISSING',
        className: 'text-rose-300 bg-rose-500/10 border-rose-500/20',
      };
    case 'provisioned':
      return {
        label: 'PROVISIONED',
        className: 'text-sky-300 bg-sky-500/10 border-sky-500/20',
      };
    case 'unknown':
      return {
        label: 'UNKNOWN',
        className: 'text-zinc-300 bg-zinc-500/10 border-zinc-500/20',
      };
    default:
      return {
        label: 'UNSET',
        className: 'text-zinc-500 bg-zinc-500/10 border-zinc-500/20',
      };
  }
}

function groupAccountsByPlatform(accounts: ConnectedAccount[]) {
  const groups = new Map<string, ConnectedAccount[]>();

  accounts.forEach((account) => {
    const current = groups.get(account.platform_id) || [];
    current.push(account);
    groups.set(account.platform_id, current);
  });

  return Array.from(groups.entries())
    .map(([platformId, groupedAccounts]) => ({
      platformId,
      accounts: groupedAccounts.sort((left, right) =>
        left.display_name.localeCompare(right.display_name),
      ),
    }))
    .sort((left, right) => left.platformId.localeCompare(right.platformId));
}

function canSelectAccount(account: ConnectedAccount) {
  if ((account.auth_method || '').toLowerCase() !== 'chromium_profile') {
    return true;
  }
  return (account.browser_status || '').toLowerCase() === 'ready';
}

export default function AccountSelector({
  accounts,
  selectedIds,
  onChange,
  emptyMessage,
  title,
  subtitle,
  selectionHint,
}: AccountSelectorProps) {
  const groups = groupAccountsByPlatform(accounts);

  const toggleAccount = (accountId: string, checked: boolean) => {
    if (checked) {
      if (selectedIds.includes(accountId)) {
        return;
      }
      onChange([...selectedIds, accountId]);
      return;
    }

    onChange(selectedIds.filter((id) => id !== accountId));
  };

  const selectMany = (ids: string[]) => {
    const next = new Set(selectedIds);
    ids.forEach((id) => next.add(id));
    onChange(Array.from(next));
  };

  const clearMany = (ids: string[]) => {
    const blocked = new Set(ids);
    onChange(selectedIds.filter((id) => !blocked.has(id)));
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-1">
        <span className="text-[10px] font-black text-zinc-500 uppercase tracking-widest">
          {title}
        </span>
        <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
          {subtitle}
        </p>
        {selectionHint && (
          <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
            {selectionHint}
          </p>
        )}
      </div>

      {groups.length > 0 ? (
        <div className="space-y-5">
          {groups.map((group) => {
            const selectableIds = group.accounts
              .filter(canSelectAccount)
              .map((account) => account.id);
            const selectedCount = group.accounts.filter((account) =>
              selectedIds.includes(account.id),
            ).length;
            const readyCount = group.accounts.filter(canSelectAccount).length;

            return (
              <section
                key={group.platformId}
                className="rounded-3xl border border-white/5 bg-black/20 p-5"
              >
                <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                  <div>
                    <p className="text-sm font-black text-white uppercase tracking-tight">
                      {formatPlatformName(group.platformId)}
                    </p>
                    <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
                      {group.accounts.length} accounts · {readyCount} ready
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => selectMany(selectableIds)}
                      disabled={selectableIds.length === 0}
                      className="rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-[10px] font-black uppercase tracking-widest text-zinc-300 transition-all hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      Select All
                    </button>
                    <button
                      type="button"
                      onClick={() => clearMany(group.accounts.map((account) => account.id))}
                      disabled={selectedCount === 0}
                      className="rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-[10px] font-black uppercase tracking-widest text-zinc-300 transition-all hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      Clear
                    </button>
                  </div>
                </div>

                <div className="grid gap-3 md:grid-cols-2">
                  {group.accounts.map((account) => {
                    const selected = selectedIds.includes(account.id);
                    const selectable = canSelectAccount(account);
                    const methodMeta = formatMethodLabel(account.auth_method);
                    const statusMeta = browserStatusMeta(account.browser_status);

                    return (
                      <label
                        key={account.id}
                        className={`flex items-start gap-4 rounded-2xl border p-4 transition-all duration-300 ${selected ? 'border-emerald-500/50 bg-emerald-500/10 shadow-lg shadow-emerald-500/5' : 'border-white/5 bg-black/30 hover:border-white/20 hover:bg-black/50'} ${!selectable ? 'opacity-55' : 'cursor-pointer'}`}
                      >
                        <input
                          type="checkbox"
                          className="sr-only"
                          checked={selected}
                          disabled={!selectable}
                          onChange={(event) =>
                            toggleAccount(account.id, event.target.checked)
                          }
                        />
                        <div
                          className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border-2 transition-all ${selected ? 'border-emerald-500 bg-emerald-500' : 'border-zinc-700'} ${!selectable ? 'border-zinc-800 bg-zinc-900' : ''}`}
                        >
                          {selected && (
                            <span className="text-xs font-black text-black">
                              ✓
                            </span>
                          )}
                        </div>

                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="truncate text-sm font-bold text-white">
                              {account.display_name}
                            </span>
                            <span
                              className={`rounded-full border px-2 py-0.5 text-[9px] font-black uppercase tracking-widest ${methodMeta.className}`}
                            >
                              {methodMeta.label}
                            </span>
                            {account.auth_method === 'chromium_profile' && (
                              <span
                                className={`rounded-full border px-2 py-0.5 text-[9px] font-black uppercase tracking-widest ${statusMeta.className}`}
                              >
                                {statusMeta.label}
                              </span>
                            )}
                          </div>
                          <p className="mt-1 text-[9px] font-black uppercase tracking-widest text-zinc-500">
                            {account.platform_id}
                          </p>
                          {!selectable && (
                            <p className="mt-2 text-[10px] font-bold uppercase tracking-widest text-amber-300">
                              Chromium profile needs login before publish
                            </p>
                          )}
                        </div>
                      </label>
                    );
                  })}
                </div>
              </section>
            );
          })}
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-white/5 bg-black/20 py-10 text-center">
          <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">
            {emptyMessage}
          </p>
        </div>
      )}
    </div>
  );
}
