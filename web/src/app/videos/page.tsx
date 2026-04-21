'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';

type DistributionJob = {
  id: number;
  generation_job_id: string;
  account_id: string;
  platform: string;
  status: 'pending' | 'uploading' | 'completed' | 'failed';
  status_detail: string;
  external_id: string;
  error: string;
  retry_source_job_id?: number | null;
  retry_attempt?: number;
  created_at: string;
  updated_at: string;
};

type VideoMetricSnapshot = {
  id: number;
  user_id: number;
  generation_job_id: string;
  distribution_job_id: number;
  account_id: string;
  platform: string;
  niche: string;
  external_id: string;
  video_title: string;
  view_count: number;
  like_count: number;
  comment_count: number;
  collected_at: string;
};

type MetricsSummary = {
  total_videos: number;
  total_views: number;
  total_likes: number;
  total_comments: number;
  latest_collected_at?: string | null;
};

type MetricsResponse = {
  summary: MetricsSummary;
  latest: VideoMetricSnapshot[];
  history: VideoMetricSnapshot[];
  alerts: PerformanceAlert[];
};

type PerformanceAlert = {
  id: string;
  level: 'medium' | 'high' | 'critical';
  platform: string;
  account_id: string;
  niche: string;
  external_id: string;
  video_title: string;
  metric: string;
  current_value: number;
  previous_value: number;
  drop_percent: number;
  message: string;
  created_at: string;
};

type CommunityReplyDraft = {
  id: number;
  user_id: number;
  generation_job_id: string;
  distribution_job_id: number;
  account_id: string;
  platform: string;
  niche: string;
  video_title: string;
  external_comment_id: string;
  parent_comment_id: string;
  comment_author: string;
  comment_text: string;
  suggested_reply: string;
  status: string;
  posted_external_id: string;
  replied_at?: string | null;
  created_at: string;
  updated_at: string;
};

type CommunitySummary = {
  total: number;
  drafts: number;
  replied: number;
  latest_created_at?: string | null;
};

type CommunityResponse = {
  summary: CommunitySummary;
  latest: CommunityReplyDraft[];
};

function formatTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('en-US').format(value || 0);
}

type TrendPoint = {
  date: string;
  value: number;
};

type TrendSeries = {
  label: string;
  total: number;
  points: TrendPoint[];
  color: string;
};

function colorForIndex(index: number) {
  const colors = ['#10b981', '#3b82f6', '#f59e0b', '#f97316', '#8b5cf6'];
  return colors[index % colors.length];
}

function buildTrendSeries(items: VideoMetricSnapshot[], keyFn: (item: VideoMetricSnapshot) => string) {
  const grouped = new Map<string, Map<string, VideoMetricSnapshot>>();

  for (const item of items) {
    const label = keyFn(item).trim();
    if (!label) continue;
    const date = item.collected_at.slice(0, 10);
    const labelMap = grouped.get(label) || new Map<string, VideoMetricSnapshot>();
    const existing = labelMap.get(`${date}:${item.external_id}`);
    if (!existing || new Date(item.collected_at).getTime() >= new Date(existing.collected_at).getTime()) {
      labelMap.set(`${date}:${item.external_id}`, item);
    }
    grouped.set(label, labelMap);
  }

  const series: TrendSeries[] = [];
  Array.from(grouped.entries()).forEach(([label, map], index) => {
    const pointsByDate = new Map<string, number>();
    map.forEach((snap) => {
      const date = snap.collected_at.slice(0, 10);
      pointsByDate.set(date, (pointsByDate.get(date) || 0) + snap.view_count);
    });
    const points = Array.from(pointsByDate.entries())
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([date, value]) => ({ date, value }));
    if (points.length === 0) return;
    series.push({
      label,
      total: points[points.length - 1]?.value || 0,
      points,
      color: colorForIndex(index),
    });
  });

  return series.sort((a, b) => b.total - a.total);
}

function Sparkline({ points, color }: { points: TrendPoint[]; color: string }) {
  if (points.length === 0) {
    return <div className="h-24 rounded-2xl border border-dashed border-white/5" />;
  }

  const width = 320;
  const height = 96;
  const min = Math.min(...points.map((p) => p.value));
  const max = Math.max(...points.map((p) => p.value));
  const span = max - min || 1;
  const step = points.length > 1 ? width / (points.length - 1) : width;
  const coords = points.map((point, index) => {
    const x = index * step;
    const y = height - ((point.value - min) / span) * (height - 16) - 8;
    return `${x},${y}`;
  });

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="h-24 w-full overflow-visible">
      <polyline
        fill="none"
        stroke={color}
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={coords.join(' ')}
      />
      {coords.map((coord, index) => {
        const [x, y] = coord.split(',');
        return <circle key={`${coord}-${index}`} cx={Number(x)} cy={Number(y)} r="3.5" fill={color} />;
      })}
    </svg>
  );
}

export default function VideoLibrary() {
  const [videos, setVideos] = useState<string[]>([]);
  const [publishHistory, setPublishHistory] = useState<DistributionJob[]>([]);
  const [metricsSummary, setMetricsSummary] = useState<MetricsSummary | null>(null);
  const [metricsLatest, setMetricsLatest] = useState<VideoMetricSnapshot[]>([]);
  const [metricsHistory, setMetricsHistory] = useState<VideoMetricSnapshot[]>([]);
  const [performanceAlerts, setPerformanceAlerts] = useState<PerformanceAlert[]>([]);
  const [communitySummary, setCommunitySummary] = useState<CommunitySummary | null>(null);
  const [communityLatest, setCommunityLatest] = useState<CommunityReplyDraft[]>([]);
  const [loading, setLoading] = useState(true);
  const [isSyncingMetrics, setIsSyncingMetrics] = useState(false);
  const [isSyncingCommunity, setIsSyncingCommunity] = useState(false);

  const nicheTrendSeries = useMemo(() => {
    return buildTrendSeries(metricsHistory, (item) => item.niche || item.platform || 'unknown').slice(0, 3);
  }, [metricsHistory]);

  const platformTrendSeries = useMemo(() => {
    return buildTrendSeries(metricsHistory, (item) => item.platform || 'unknown').slice(0, 3);
  }, [metricsHistory]);

  const accountTrendSeries = useMemo(() => {
    return buildTrendSeries(metricsHistory, (item) => item.account_id || 'unknown').slice(0, 3);
  }, [metricsHistory]);

  const videoTrendSeries = useMemo(() => {
    return buildTrendSeries(metricsHistory, (item) => item.video_title || item.external_id || 'unknown').slice(0, 3);
  }, [metricsHistory]);

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await fetch('/api/metrics', { credentials: 'include' });
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as MetricsResponse;
      setMetricsSummary(data.summary || null);
      setMetricsLatest(data.latest || []);
      setMetricsHistory(data.history || []);
      setPerformanceAlerts(data.alerts || []);
    } catch (err) {
      console.error('Failed to fetch metrics:', err);
    }
  }, []);

  const fetchCommunity = useCallback(async () => {
    try {
      const res = await fetch('/api/community/replies', { credentials: 'include' });
      if (!res.ok) {
        return;
      }
      const data = (await res.json()) as CommunityResponse;
      setCommunitySummary(data.summary || null);
      setCommunityLatest(data.latest || []);
    } catch (err) {
      console.error('Failed to fetch community replies:', err);
    }
  }, []);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [videosRes, historyRes] = await Promise.all([
          fetch('/api/videos'),
          fetch('/api/publish/history')
        ]);

        if (videosRes.ok) {
          const data = await videosRes.json();
          setVideos(data || []);
        }

        if (historyRes.ok) {
          const data = await historyRes.json();
          setPublishHistory(data || []);
        }

        await Promise.all([fetchMetrics(), fetchCommunity()]);
      } catch (err) {
        console.error('Failed to fetch data:', err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [fetchCommunity, fetchMetrics]);

  const syncMetrics = async () => {
    try {
      setIsSyncingMetrics(true);
      const res = await fetch('/api/metrics/sync', {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      await fetchMetrics();
    } catch (err) {
      console.error('Failed to sync metrics:', err);
    } finally {
      setIsSyncingMetrics(false);
    }
  };

  const syncCommunity = async () => {
    try {
      setIsSyncingCommunity(true);
      const res = await fetch('/api/community/sync', {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      await fetchCommunity();
    } catch (err) {
      console.error('Failed to sync community:', err);
    } finally {
      setIsSyncingCommunity(false);
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-20">
      <div className="flex flex-col md:flex-row justify-between items-center gap-6">
        <div className="flex flex-col sm:flex-row gap-3 ml-auto">
          <button
            onClick={syncMetrics}
            disabled={isSyncingMetrics}
            className="rounded-2xl bg-emerald-500/10 border border-emerald-500/20 px-5 py-3 text-[10px] font-black uppercase tracking-widest text-emerald-300 transition-all hover:bg-emerald-500/20 active:scale-95 disabled:opacity-50"
          >
            {isSyncingMetrics ? 'SYNCING METRICS...' : 'SYNC METRICS'}
          </button>
          <button
            onClick={syncCommunity}
            disabled={isSyncingCommunity}
            className="rounded-2xl bg-fuchsia-500/10 border border-fuchsia-500/20 px-5 py-3 text-[10px] font-black uppercase tracking-widest text-fuchsia-300 transition-all hover:bg-fuchsia-500/20 active:scale-95 disabled:opacity-50"
          >
            {isSyncingCommunity ? 'SYNCING COMMUNITY...' : 'SYNC COMMUNITY'}
          </button>
          <div className="bg-zinc-900 border border-white/5 rounded-2xl px-6 py-3 text-xs font-bold text-zinc-500">
              TOTAL ASSETS: {videos.length}
          </div>
        </div>
      </div>

      <section className="grid gap-4 md:grid-cols-4">
        <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
          <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Tracked Videos</p>
          <p className="mt-2 text-3xl font-black text-white">{metricsSummary ? formatNumber(metricsSummary.total_videos) : '0'}</p>
        </div>
        <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
          <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Total Views</p>
          <p className="mt-2 text-3xl font-black text-emerald-400">{metricsSummary ? formatNumber(metricsSummary.total_views) : '0'}</p>
        </div>
        <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
          <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Total Likes</p>
          <p className="mt-2 text-3xl font-black text-blue-400">{metricsSummary ? formatNumber(metricsSummary.total_likes) : '0'}</p>
        </div>
        <div className="rounded-3xl border border-white/5 bg-card p-6 shadow-xl">
          <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Total Comments</p>
          <p className="mt-2 text-3xl font-black text-amber-400">{metricsSummary ? formatNumber(metricsSummary.total_comments) : '0'}</p>
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-2">
        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Growth Chart</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Views by Niche</h3>
          </div>
          <div className="space-y-5">
            {nicheTrendSeries.length > 0 ? (
              nicheTrendSeries.map((series) => (
                <div key={series.label} className="rounded-2xl border border-white/5 bg-black/40 p-4">
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight">{series.label}</p>
                      <p className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">{formatNumber(series.total)} total views</p>
                    </div>
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: series.color }} />
                  </div>
                  <Sparkline points={series.points} color={series.color} />
                </div>
              ))
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No niche trends yet</p>
                <p className="mt-2 text-zinc-500 text-sm">Sync metrics after you have published videos with connected accounts.</p>
              </div>
            )}
          </div>
        </div>

        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Growth Chart</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Views by Video</h3>
          </div>
          <div className="space-y-5">
            {videoTrendSeries.length > 0 ? (
              videoTrendSeries.map((series) => (
                <div key={series.label} className="rounded-2xl border border-white/5 bg-black/40 p-4">
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight line-clamp-1">{series.label}</p>
                      <p className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">{formatNumber(series.total)} total views</p>
                    </div>
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: series.color }} />
                  </div>
                  <Sparkline points={series.points} color={series.color} />
                </div>
              ))
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No video trends yet</p>
              </div>
            )}
          </div>
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-2">
        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Growth Chart</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Views by Platform</h3>
          </div>
          <div className="space-y-5">
            {platformTrendSeries.length > 0 ? (
              platformTrendSeries.map((series) => (
                <div key={series.label} className="rounded-2xl border border-white/5 bg-black/40 p-4">
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight">{series.label}</p>
                      <p className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">{formatNumber(series.total)} total views</p>
                    </div>
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: series.color }} />
                  </div>
                  <Sparkline points={series.points} color={series.color} />
                </div>
              ))
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No platform trends yet</p>
              </div>
            )}
          </div>
        </div>

        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Growth Chart</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Views by Account</h3>
          </div>
          <div className="space-y-5">
            {accountTrendSeries.length > 0 ? (
              accountTrendSeries.map((series) => (
                <div key={series.label} className="rounded-2xl border border-white/5 bg-black/40 p-4">
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight line-clamp-1">{series.label}</p>
                      <p className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">{formatNumber(series.total)} total views</p>
                    </div>
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: series.color }} />
                  </div>
                  <Sparkline points={series.points} color={series.color} />
                </div>
              ))
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No account trends yet</p>
              </div>
            )}
          </div>
        </div>
      </section>

      {/* Account Performance Leaderboard (Comparison View) */}
      <section className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between gap-4 mb-8">
          <div>
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Leaderboard</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Account Comparison</h3>
          </div>
          <span className="text-[10px] font-black text-zinc-600 uppercase tracking-widest bg-black/40 px-4 py-2 rounded-xl border border-white/5">
            Live Metrics
          </span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left border-separate border-spacing-y-3">
            <thead>
              <tr className="text-[10px] font-black text-zinc-500 uppercase tracking-widest">
                <th className="px-6 py-4">Account ID</th>
                <th className="px-6 py-4">Platform</th>
                <th className="px-4 py-4 text-center">Relative Reach</th>
                <th className="px-6 py-4 text-right">Total Impact</th>
              </tr>
            </thead>
            <tbody>
              {accountTrendSeries.length > 0 ? (
                accountTrendSeries.map((series) => {
                  const sample = metricsHistory.find(m => m.account_id === series.label);
                  const platform = sample?.platform || 'unknown';
                  return (
                    <tr key={series.label} className="group/row bg-black/40 hover:bg-black/60 transition-all duration-300">
                      <td className="px-6 py-5 rounded-l-2xl border-y border-l border-white/5">
                        <span className="text-sm font-black text-white group-hover/row:text-blue-400 transition-colors">{series.label}</span>
                      </td>
                      <td className="px-6 py-5 border-y border-white/5">
                        <span className="text-[10px] font-bold text-zinc-500 uppercase bg-zinc-900 px-3 py-1 rounded-lg border border-white/5">
                          {platform}
                        </span>
                      </td>
                      <td className="px-4 py-5 border-y border-white/5 text-center">
                        <div className="flex items-center gap-4 max-w-[200px] mx-auto">
                          <div className="flex-1 bg-zinc-900 h-1.5 rounded-full overflow-hidden">
                            <div className="bg-gradient-to-r from-blue-600 to-blue-400 h-full shadow-[0_0_8px_rgba(59,130,246,0.4)]" style={{ width: `${Math.min(100, (series.total / (metricsSummary?.total_views || 1)) * 100)}%` }}></div>
                          </div>
                          <span className="text-[10px] font-mono font-bold text-zinc-500 w-10">
                            {((series.total / (metricsSummary?.total_views || 1)) * 100).toFixed(1)}%
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-5 rounded-r-2xl border-y border-r border-white/5 text-right">
                        <span className="text-sm font-black text-emerald-400 font-mono tracking-tighter">
                          {formatNumber(series.total)} views
                        </span>
                      </td>
                    </tr>
                  );
                })
              ) : (
                <tr>
                  <td colSpan={4} className="py-20 text-center text-zinc-600 font-bold uppercase tracking-widest text-[10px] border-2 border-dashed border-white/5 rounded-[2rem]">
                    Cross-account comparison pending...
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-[1.15fr_0.85fr]">
        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="flex items-center justify-between gap-4 mb-6">
            <div>
              <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Analytics</p>
              <h3 className="text-2xl font-black text-white uppercase tracking-tight">Latest YouTube Metrics</h3>
            </div>
            <div className="text-right">
              <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Last Sync</p>
              <p className="text-xs font-bold text-zinc-300">{metricsSummary?.latest_collected_at ? formatTime(metricsSummary.latest_collected_at) : 'Never'}</p>
            </div>
          </div>

          {metricsLatest.length > 0 ? (
            <div className="grid gap-4">
              {metricsLatest.map((metric) => (
                <div key={metric.id} className="rounded-2xl border border-white/5 bg-black/40 p-5">
                  <div className="flex items-start justify-between gap-4 mb-3">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight line-clamp-2">{metric.video_title || metric.external_id}</p>
                      <p className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest mt-1">
                        {metric.platform} · {metric.account_id}
                      </p>
                    </div>
                    <span className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 px-2 py-1 text-[9px] font-black uppercase tracking-widest text-emerald-300">
                      {formatTime(metric.collected_at)}
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-3">
                    <div className="rounded-xl border border-white/5 bg-black/30 p-3">
                      <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Views</p>
                      <p className="mt-1 text-lg font-black text-emerald-400">{formatNumber(metric.view_count)}</p>
                    </div>
                    <div className="rounded-xl border border-white/5 bg-black/30 p-3">
                      <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Likes</p>
                      <p className="mt-1 text-lg font-black text-blue-400">{formatNumber(metric.like_count)}</p>
                    </div>
                    <div className="rounded-xl border border-white/5 bg-black/30 p-3">
                      <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Comments</p>
                      <p className="mt-1 text-lg font-black text-amber-400">{formatNumber(metric.comment_count)}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No synced metrics yet</p>
                <p className="mt-2 text-zinc-500 text-sm">Use Sync Metrics after the first publish batch lands in YouTube API.</p>
              </div>
            )}
        </div>

        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Snapshot History</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Recent Samples</h3>
          </div>
          <div className="space-y-3 max-h-[620px] overflow-auto pr-2 custom-scrollbar">
            {metricsHistory.length > 0 ? (
              metricsHistory.slice(0, 20).map((snap) => (
                <div key={snap.id} className="rounded-2xl border border-white/5 bg-black/40 p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight">{snap.video_title || snap.external_id}</p>
                      <p className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">
                        {snap.platform} · {snap.account_id}
                      </p>
                    </div>
                    <span className="text-[10px] font-black uppercase tracking-widest text-zinc-500">
                      {formatTime(snap.collected_at)}
                    </span>
                  </div>
                  <div className="mt-3 grid grid-cols-3 gap-2 text-[10px] font-black uppercase tracking-widest">
                    <div className="rounded-xl border border-white/5 bg-black/30 p-3 text-emerald-400">V {formatNumber(snap.view_count)}</div>
                    <div className="rounded-xl border border-white/5 bg-black/30 p-3 text-blue-400">L {formatNumber(snap.like_count)}</div>
                    <div className="rounded-xl border border-white/5 bg-black/30 p-3 text-amber-400">C {formatNumber(snap.comment_count)}</div>
                  </div>
                </div>
              ))
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No metric snapshots yet</p>
              </div>
            )}
          </div>
        </div>
      </section>

      <section className="grid gap-8 lg:grid-cols-[0.8fr_1.2fr]">
        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Community Agent</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Comment Reply Drafts</h3>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
              <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Total</p>
              <p className="mt-1 text-2xl font-black text-white">{communitySummary ? formatNumber(communitySummary.total) : '0'}</p>
            </div>
            <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
              <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Drafts</p>
              <p className="mt-1 text-2xl font-black text-fuchsia-300">{communitySummary ? formatNumber(communitySummary.drafts) : '0'}</p>
            </div>
            <div className="rounded-2xl border border-white/5 bg-black/30 p-4">
              <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Replied</p>
              <p className="mt-1 text-2xl font-black text-emerald-400">{communitySummary ? formatNumber(communitySummary.replied) : '0'}</p>
            </div>
          </div>
          <p className="mt-4 text-[10px] font-black uppercase tracking-widest text-zinc-500">
            Last Draft Sync: {communitySummary?.latest_created_at ? formatTime(communitySummary.latest_created_at) : 'Never'}
          </p>
        </div>

        <div className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
          <div className="mb-6">
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Community Queue</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Latest Reply Suggestions</h3>
          </div>
          <div className="space-y-4 max-h-[420px] overflow-auto pr-2 custom-scrollbar">
            {communityLatest.length > 0 ? (
              communityLatest.slice(0, 10).map((draft) => (
                <div key={draft.id} className="rounded-2xl border border-white/5 bg-black/40 p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-sm font-black text-white uppercase tracking-tight line-clamp-1">{draft.video_title || draft.external_comment_id}</p>
                      <p className="mt-1 text-[10px] font-mono text-zinc-500 uppercase tracking-widest">
                        {draft.platform} · {draft.comment_author || 'viewer'}
                      </p>
                    </div>
                    <span className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest ${
                      draft.status === 'replied'
                        ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-300'
                        : 'border-fuchsia-500/20 bg-fuchsia-500/10 text-fuchsia-300'
                    }`}>
                      {draft.status}
                    </span>
                  </div>
                  <p className="mt-3 text-sm text-zinc-300 line-clamp-3">{draft.comment_text}</p>
                  <div className="mt-3 rounded-xl border border-white/5 bg-black/30 p-3">
                    <p className="text-[9px] font-black uppercase tracking-widest text-zinc-500">Suggested Reply</p>
                    <p className="mt-1 text-sm text-white line-clamp-4">{draft.suggested_reply}</p>
                  </div>
                  <p className="mt-3 text-[10px] font-black uppercase tracking-widest text-zinc-600">
                    {formatTime(draft.created_at)}
                  </p>
                </div>
              ))
            ) : (
              <div className="rounded-3xl border-2 border-dashed border-white/5 py-16 text-center">
                <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No reply drafts yet</p>
              </div>
            )}
          </div>
        </div>
      </section>

      <section className="rounded-[2.5rem] border border-white/5 bg-card p-8 shadow-2xl">
        <div className="flex items-center justify-between gap-4 mb-6">
          <div>
            <p className="text-[10px] font-black uppercase tracking-widest text-zinc-500">Performance Watch</p>
            <h3 className="text-2xl font-black text-white uppercase tracking-tight">Sharp Drop Alerts</h3>
          </div>
          <span className="rounded-full border border-white/5 bg-black/30 px-3 py-1 text-[10px] font-black uppercase tracking-widest text-zinc-400">
            {performanceAlerts.length} active
          </span>
        </div>

        {performanceAlerts.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2">
            {performanceAlerts.map((alert) => (
              <div
                key={alert.id}
                className={`rounded-2xl border p-4 ${
                  alert.level === 'critical'
                    ? 'border-red-500/30 bg-red-500/10'
                    : alert.level === 'high'
                      ? 'border-orange-500/30 bg-orange-500/10'
                      : 'border-amber-500/30 bg-amber-500/10'
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-sm font-black text-white uppercase tracking-tight line-clamp-1">{alert.video_title || alert.external_id}</p>
                    <p className="mt-1 text-[10px] font-mono text-zinc-300 uppercase tracking-widest">
                      {alert.platform} · {alert.account_id}
                    </p>
                  </div>
                  <span className="rounded-lg border border-white/10 bg-black/30 px-2 py-1 text-[9px] font-black uppercase tracking-widest text-white">
                    -{alert.drop_percent.toFixed(1)}%
                  </span>
                </div>
                <p className="mt-3 text-sm text-zinc-100">{alert.message}</p>
                <div className="mt-3 grid grid-cols-2 gap-3 text-[10px] font-black uppercase tracking-widest">
                  <div className="rounded-xl border border-white/5 bg-black/30 p-3 text-emerald-300">Current {formatNumber(alert.current_value)}</div>
                  <div className="rounded-xl border border-white/5 bg-black/30 p-3 text-blue-300">Previous {formatNumber(alert.previous_value)}</div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="rounded-3xl border-2 border-dashed border-white/5 py-14 text-center">
            <p className="text-zinc-600 text-sm font-bold uppercase tracking-widest">No performance alerts yet</p>
          </div>
        )}
      </section>

      {loading ? (
        <div className="flex justify-center py-32">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 shadow-[0_0_20px_rgba(59,130,246,0.3)]"></div>
        </div>
      ) : (
        <div className="grid gap-12 lg:grid-cols-[1fr_1fr]">
          {/* Raw Assets Column */}
          <div className="space-y-6">
            <div className="flex items-center gap-3 mb-2">
               <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center text-lg shadow-lg border border-blue-500/30 text-blue-400">📂</div>
               <h3 className="text-xl font-bold text-white uppercase tracking-tight">Raw Assets</h3>
            </div>
            <div className="grid gap-4 max-h-[800px] overflow-auto pr-2 custom-scrollbar">
              {videos.length > 0 ? (
                videos.map((video) => (
                  <div key={video} className="bg-card border border-white/5 rounded-2xl p-6 flex items-center justify-between hover:border-blue-500/30 transition-all duration-300 group shadow-xl">
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 bg-zinc-900 rounded-xl flex items-center justify-center text-2xl shadow-inner border border-white/5 group-hover:scale-110 transition-transform duration-500">
                        🎬
                      </div>
                      <div className="max-w-[200px] md:max-w-xs">
                        <p className="font-mono text-xs font-bold text-white mb-1 group-hover:text-blue-400 transition-colors uppercase tracking-tighter truncate">{video}</p>
                        <span className="text-[9px] font-black text-zinc-600 bg-black/40 px-2 py-0.5 rounded border border-white/5 uppercase tracking-widest">MP4 CONTAINER</span>
                      </div>
                    </div>
                    <div className="flex gap-2">
                       <button className="p-3 bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-white rounded-lg text-[10px] font-black transition-all border border-white/5 active:scale-95 uppercase tracking-widest">
                        DL
                      </button>
                      <button className="px-4 py-3 bg-blue-600/10 text-blue-400 border border-blue-500/20 hover:bg-blue-600 hover:text-white rounded-lg text-[10px] font-black transition-all active:scale-95 uppercase tracking-widest">
                        Preview
                      </button>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-20 bg-card rounded-3xl border-2 border-dashed border-white/5">
                  <p className="text-zinc-600 font-bold uppercase tracking-widest text-xs">No raw asset files yet</p>
                </div>
              )}
            </div>
          </div>

          {/* Publish History Column */}
          <div className="space-y-6">
            <div className="flex items-center gap-3 mb-2">
               <div className="w-8 h-8 bg-emerald-500/20 rounded-lg flex items-center justify-center text-lg shadow-lg border border-emerald-500/30 text-emerald-400">🌐</div>
               <h3 className="text-xl font-bold text-white uppercase tracking-tight">Publish History</h3>
            </div>
            <div className="grid gap-4 max-h-[800px] overflow-auto pr-2 custom-scrollbar">
              {publishHistory.length > 0 ? (
                publishHistory.map((dist) => (
                  <div key={dist.id} className="bg-card border border-white/5 rounded-2xl p-6 hover:border-emerald-500/30 transition-all duration-300 group shadow-xl">
                    <div className="flex items-center justify-between mb-4">
                        <div className="flex items-center gap-3">
                            <div className="w-10 h-10 bg-black/40 rounded-xl flex items-center justify-center border border-white/5 group-hover:border-emerald-500/30 transition-colors">
                                <span className="text-xs font-black text-zinc-400 uppercase">{dist.platform.slice(0, 2)}</span>
                            </div>
                            <div>
                                <p className="text-sm font-black text-white uppercase tracking-tight">{dist.platform}</p>
                                <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">Account: {dist.account_id}</p>
                            </div>
                        </div>
                        <span className={`rounded-lg border px-2 py-1 text-[9px] font-black uppercase tracking-widest ${
                            dist.status === 'completed' ? 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30' : 
                            dist.status === 'failed' ? 'bg-red-500/15 text-red-300 border-red-500/30' : 
                            'bg-amber-500/15 text-amber-300 border-amber-500/30'
                        }`}>
                            {dist.status}
                        </span>
                    </div>

                    <div className="grid grid-cols-2 gap-4 bg-black/40 rounded-xl p-4 border border-white/5">
                        <div className="space-y-1">
                            <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">External ID</p>
                            <p className="text-[10px] text-zinc-300 font-mono break-all">{dist.external_id || 'N/A'}</p>
                        </div>
                        <div className="space-y-1">
                            <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">Job ID</p>
                            <p className="text-[10px] text-zinc-300 font-mono break-all">{dist.generation_job_id}</p>
                        </div>
                        <div className="space-y-1 col-span-2">
                            <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">Last Update</p>
                            <p className="text-[10px] text-zinc-400 font-bold uppercase">{formatTime(dist.updated_at)}</p>
                        </div>
                        {(dist.retry_attempt || dist.retry_source_job_id) && (
                            <div className="space-y-1 col-span-2">
                                <p className="text-[9px] text-zinc-600 font-black uppercase tracking-widest">Retry Chain</p>
                                <p className="text-[10px] text-zinc-400 font-bold uppercase">
                                    Attempt {dist.retry_attempt || 0}
                                    {dist.retry_source_job_id ? ` · failover from ${dist.retry_source_job_id}` : ''}
                                </p>
                            </div>
                        )}
                        {dist.error && (
                            <div className="col-span-2 mt-2 bg-red-500/5 border border-red-500/10 rounded-lg p-2">
                                <p className="text-[9px] text-red-400 font-bold uppercase tracking-widest mb-1">Error Report</p>
                                <p className="text-[10px] text-red-300/80 font-medium italic">{dist.error}</p>
                            </div>
                        )}
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-20 bg-card rounded-3xl border-2 border-dashed border-white/5">
                  <p className="text-zinc-600 font-bold uppercase tracking-widest text-xs">No publish records yet</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
