import React, { useEffect, useState, useCallback, useMemo } from 'react';
import {
  Cloud,
  Server,
  HardDrive,
  AlertTriangle,
  CheckCircle2,
  Activity,
  Cpu,
  Network,
  Shield,
  Database,
  RefreshCw,
  ArrowRight,
  TrendingUp,
  CircleDot,
  Layers,
  Wifi,
  WifiOff,
  AlertCircle,
  BarChart3,
  Bot,
  Clock,
  Zap,
  MonitorSpeaker,
  Gauge,
} from 'lucide-react';
import { getResourceMonitor } from '../services/api';
import useStore from '../store/useStore';

// Auto-refresh interval (30 seconds)
const REFRESH_INTERVAL = 30000;

// Simulated time-series data (last 12 data points) for mini-chart
function useMiniChart(currentValue, maxPoints = 12) {
  const [history, setHistory] = useState([]);
  useEffect(() => {
    if (currentValue == null || currentValue === '--') return;
    setHistory((prev) => {
      const next = [...prev, currentValue];
      if (next.length > maxPoints) return next.slice(-maxPoints);
      return next;
    });
  }, [currentValue, maxPoints]);
  return history;
}

// Simple CSS bar-chart component
function MiniBarChart({ data, color = '#513CC8', height = 32 }) {
  if (!data || data.length === 0) return null;
  const max = Math.max(...data, 1);
  return (
    <div className="flex items-end gap-px" style={{ height }}>
      {data.map((v, i) => (
        <div
          key={i}
          className="flex-1 rounded-t-sm transition-all duration-500"
          style={{
            height: `${Math.max((v / max) * 100, 8)}%`,
            background: i === data.length - 1 ? color : `${color}60`,
            minWidth: 3,
          }}
        />
      ))}
    </div>
  );
}

// Donut/ring progress component
function RingProgress({ value, max, size = 64, strokeWidth = 6, color = '#513CC8', label }) {
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const pct = max > 0 ? Math.min(value / max, 1) : 0;
  const offset = circumference * (1 - pct);
  return (
    <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={radius} stroke="#e5e7eb" strokeWidth={strokeWidth} fill="none" />
        <circle
          cx={size / 2} cy={size / 2} r={radius}
          stroke={color} strokeWidth={strokeWidth} fill="none"
          strokeDasharray={circumference} strokeDashoffset={offset}
          strokeLinecap="round"
          className="transition-all duration-700"
        />
      </svg>
      <div className="absolute flex flex-col items-center">
        <span className="text-sm font-bold text-gray-800">{value}</span>
        {label && <span className="text-[10px] text-gray-400">{label}</span>}
      </div>
    </div>
  );
}

export default function ResourceMonitorPage() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [lastRefresh, setLastRefresh] = useState(null);
  const [fullscreen, setFullscreen] = useState(false);
  const setActivePage = useStore((s) => s.setActivePage);

  const loadData = useCallback(async (isRefresh = false) => {
    try {
      if (isRefresh) setRefreshing(true);
      const res = await getResourceMonitor();
      if (res.code === 0) {
        setData(res.data);
        setLastRefresh(new Date());
      }
    } catch (err) {
      console.error('Failed to load resource monitor data:', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadData();
    const timer = setInterval(() => loadData(true), REFRESH_INTERVAL);
    return () => clearInterval(timer);
  }, [loadData]);

  // Mini-chart data histories
  const vmHistory = useMiniChart(data?.total_vms);
  const volumeHistory = useMiniChart(data?.total_volumes);
  const alertHistory = useMiniChart(data?.firing_alerts);

  // Time display
  const [now, setNow] = useState(new Date());
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(t);
  }, []);

  // ============ Summary cards ============
  const summaryCards = [
    {
      label: '云平台数',
      value: data?.cloud_platforms ?? '--',
      icon: Cloud,
      gradient: 'from-blue-500 to-blue-600',
      bg: 'bg-blue-50',
      ring: 'ring-blue-200',
      desc: '已接入平台',
      color: '#3b82f6',
    },
    {
      label: '虚拟机数',
      value: data?.total_vms ?? '--',
      icon: Server,
      gradient: 'from-violet-500 to-purple-600',
      bg: 'bg-violet-50',
      ring: 'ring-violet-200',
      desc: '运行中实例',
      color: '#7c3aed',
      chart: vmHistory,
    },
    {
      label: '云硬盘数',
      value: data?.total_volumes ?? '--',
      icon: HardDrive,
      gradient: 'from-emerald-500 to-green-600',
      bg: 'bg-emerald-50',
      ring: 'ring-emerald-200',
      desc: '块存储卷',
      color: '#10b981',
      chart: volumeHistory,
    },
    {
      label: '正在告警',
      value: data?.firing_alerts ?? 0,
      icon: AlertTriangle,
      gradient: data?.firing_alerts > 0 ? 'from-red-500 to-rose-600' : 'from-gray-400 to-gray-500',
      bg: data?.firing_alerts > 0 ? 'bg-red-50' : 'bg-gray-50',
      ring: data?.firing_alerts > 0 ? 'ring-red-200' : 'ring-gray-200',
      desc: '活跃告警',
      pulse: data?.firing_alerts > 0,
      color: data?.firing_alerts > 0 ? '#ef4444' : '#9ca3af',
      chart: alertHistory,
    },
    {
      label: '已恢复告警',
      value: data?.resolved_alerts ?? 0,
      icon: CheckCircle2,
      gradient: 'from-teal-500 to-cyan-600',
      bg: 'bg-teal-50',
      ring: 'ring-teal-200',
      desc: '已自动恢复',
      color: '#14b8a6',
    },
    {
      label: 'AI 智能体',
      value: data?.agents ?? '--',
      icon: Bot,
      gradient: 'from-amber-500 to-orange-600',
      bg: 'bg-amber-50',
      ring: 'ring-amber-200',
      desc: '运行中 Agent',
      color: '#f59e0b',
    },
  ];

  // ============ Component health icon map ============
  const componentIconMap = {
    '认证服务 (Keystone)': Shield,
    '计算服务 (Nova)': Server,
    '存储服务 (Cinder)': Database,
    '网络服务 (Neutron)': Network,
    '负载均衡 (Octavia)': Layers,
    '监控服务 (ECMS)': BarChart3,
  };

  const statusColor = (status) => {
    switch (status) {
      case 'healthy': return { dot: 'bg-emerald-500', text: 'text-emerald-700', bg: 'bg-emerald-50', border: 'border-emerald-200', label: '正常' };
      case 'degraded': return { dot: 'bg-yellow-500', text: 'text-yellow-700', bg: 'bg-yellow-50', border: 'border-yellow-200', label: '降级' };
      case 'down': return { dot: 'bg-red-500', text: 'text-red-700', bg: 'bg-red-50', border: 'border-red-200', label: '故障' };
      default: return { dot: 'bg-gray-400', text: 'text-gray-500', bg: 'bg-gray-50', border: 'border-gray-200', label: '未知' };
    }
  };

  const platformStatusBadge = (status) => {
    switch (status) {
      case 'connected': return { color: 'text-emerald-700', bg: 'bg-emerald-50', border: 'border-emerald-200', icon: Wifi, label: '已连接' };
      case 'failed': return { color: 'text-red-700', bg: 'bg-red-50', border: 'border-red-200', icon: WifiOff, label: '连接失败' };
      default: return { color: 'text-gray-500', bg: 'bg-gray-50', border: 'border-gray-200', icon: CircleDot, label: '未测试' };
    }
  };

  // Compute overall health score
  const healthScore = useMemo(() => {
    if (!data?.components) return null;
    const total = data.components.length;
    if (total === 0) return null;
    const healthy = data.components.filter(c => c.status === 'healthy').length;
    return { healthy, total, pct: Math.round((healthy / total) * 100) };
  }, [data]);

  // ============ Shimmer loading placeholders ============
  const ShimmerCard = () => (
    <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5 animate-pulse">
      <div className="flex items-center gap-3 mb-4">
        <div className="w-11 h-11 bg-gray-100 rounded-xl" />
        <div className="h-4 w-20 bg-gray-100 rounded" />
      </div>
      <div className="h-8 w-16 bg-gray-100 rounded mb-2" />
      <div className="h-3 w-24 bg-gray-50 rounded" />
    </div>
  );

  return (
    <div className={`h-full overflow-y-auto ${fullscreen ? 'fixed inset-0 z-50 bg-gray-50' : ''}`} style={{ scrollbarWidth: 'thin' }}>
      <div className="p-6 space-y-6 max-w-[1600px] mx-auto">

        {/* ===== Header with refresh ===== */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <div className="w-1.5 h-6 rounded-full" style={{ background: '#513CC8' }} />
              <h2 className="text-xl font-bold text-gray-800">资源监控大屏</h2>
              {healthScore && (
                <span className={`ml-2 flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border font-medium ${
                  healthScore.pct === 100
                    ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                    : healthScore.pct >= 50
                      ? 'bg-yellow-50 text-yellow-700 border-yellow-200'
                      : 'bg-red-50 text-red-700 border-red-200'
                }`}>
                  <Gauge className="w-3 h-3" />
                  健康评分 {healthScore.pct}%
                </span>
              )}
            </div>
            <p className="text-sm text-gray-400 mt-1 ml-3.5">实时监控所有接入平台的资源状态和告警信息</p>
          </div>
          <div className="flex items-center gap-3">
            {/* Live clock */}
            <div className="flex items-center gap-1.5 text-xs text-gray-400 bg-gray-50 px-3 py-1.5 rounded-lg border border-gray-100">
              <Clock className="w-3 h-3" />
              {now.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
            </div>
            {lastRefresh && (
              <span className="text-xs text-gray-400">
                上次刷新: {lastRefresh.toLocaleTimeString('zh-CN')}
              </span>
            )}
            <button
              onClick={() => loadData(true)}
              disabled={refreshing}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg transition-all border border-gray-200 hover:border-[#513CC8] hover:text-[#513CC8] hover:bg-[#EEE9FB] disabled:opacity-50"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
              刷新
            </button>
          </div>
        </div>

        {/* ===== Summary Cards Row ===== */}
        {loading ? (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            {[1,2,3,4,5,6].map(i => <ShimmerCard key={i} />)}
          </div>
        ) : (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            {summaryCards.map((card, i) => {
              const Icon = card.icon;
              return (
                <div
                  key={i}
                  className="relative bg-white rounded-2xl border border-gray-100 shadow-sm p-5 hover:shadow-lg transition-all duration-300 overflow-hidden group"
                >
                  {/* Decorative gradient accent */}
                  <div className={`absolute top-0 left-0 right-0 h-1 bg-gradient-to-r ${card.gradient}`} />
                  <div className="flex items-center gap-3 mb-3">
                    <div className={`w-11 h-11 rounded-xl flex items-center justify-center ${card.bg} ring-1 ${card.ring} group-hover:scale-110 transition-transform`}>
                      <Icon className="w-5 h-5 text-gray-600" />
                    </div>
                    <span className="text-xs text-gray-400 font-medium">{card.label}</span>
                  </div>
                  <div className="flex items-end gap-2">
                    <p className="text-3xl font-bold text-gray-800 leading-none">
                      {card.value}
                    </p>
                    {card.pulse && (
                      <span className="relative flex h-2.5 w-2.5 mb-1">
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
                        <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-red-500" />
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-gray-400 mt-1.5">{card.desc}</p>
                  {/* Mini trend chart */}
                  {card.chart && card.chart.length > 1 && (
                    <div className="mt-3">
                      <MiniBarChart data={card.chart} color={card.color} height={24} />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* ===== Middle Section: Alerts + Platform Resources ===== */}
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">

          {/* --- Alert Panel (3/5) --- */}
          <div className="lg:col-span-3 bg-white rounded-2xl border border-gray-100 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-50 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <AlertTriangle className="w-4.5 h-4.5 text-amber-500" />
                <div>
                  <h3 className="text-base font-semibold text-gray-800">告警信息</h3>
                  <p className="text-xs text-gray-400 mt-0.5">
                    {data?.firing_alerts > 0
                      ? `${data.firing_alerts} 条活跃告警`
                      : '当前无活跃告警'}
                  </p>
                </div>
              </div>
              {(data?.firing_alerts > 0 || data?.resolved_alerts > 0) && (
                <div className="flex items-center gap-2">
                  <span className="flex items-center gap-1 text-xs px-2 py-1 rounded-full bg-red-50 text-red-600 border border-red-100">
                    <AlertCircle className="w-3 h-3" /> 告警中 {data?.firing_alerts || 0}
                  </span>
                  <span className="flex items-center gap-1 text-xs px-2 py-1 rounded-full bg-emerald-50 text-emerald-600 border border-emerald-100">
                    <CheckCircle2 className="w-3 h-3" /> 已恢复 {data?.resolved_alerts || 0}
                  </span>
                </div>
              )}
            </div>
            <div className="p-6">
              {loading ? (
                <div className="space-y-3">
                  {[1,2,3].map(i => (
                    <div key={i} className="h-14 bg-gray-50 rounded-xl animate-pulse" />
                  ))}
                </div>
              ) : (data?.alerts && data.alerts.length > 0) ? (
                <div className="max-h-96 overflow-y-auto rounded-xl border border-gray-100" style={{ scrollbarWidth: 'thin' }}>
                  <table className="w-full text-sm">
                    <thead className="sticky top-0 z-10">
                      <tr className="bg-gray-50 border-b border-gray-100">
                        <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">状态</th>
                        <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">已接入云平台</th>
                        <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">告警优先级</th>
                        <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">告警内容</th>
                        <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">告警对象</th>
                        <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">时间</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-50">
                      {data.alerts.map((alert, i) => (
                        <tr
                          key={i}
                          className={`transition-colors ${
                            alert.state === 'firing'
                              ? 'bg-red-50/30 hover:bg-red-50/60'
                              : 'bg-emerald-50/20 hover:bg-emerald-50/40'
                          }`}
                        >
                          {/* Status icon */}
                          <td className="px-4 py-3">
                            {alert.state === 'firing' ? (
                              <div className="relative inline-flex">
                                <AlertTriangle className="w-4.5 h-4.5 text-red-500" />
                                <span className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-red-500 rounded-full animate-ping" />
                              </div>
                            ) : (
                              <CheckCircle2 className="w-4.5 h-4.5 text-emerald-500" />
                            )}
                          </td>
                          {/* Cloud platform */}
                          <td className="px-4 py-3">
                            <div className="flex items-center gap-1.5">
                              <Cloud className="w-3.5 h-3.5 text-blue-400" />
                              <span className="text-sm text-gray-700 font-medium">{alert.platform || '-'}</span>
                            </div>
                          </td>
                          {/* Severity */}
                          <td className="px-4 py-3">
                            {alert.severity ? (
                              <span className={`inline-flex items-center gap-1 text-xs font-medium px-2 py-1 rounded-full ${
                                alert.severity === 'critical'
                                  ? 'bg-red-100 text-red-700 border border-red-200'
                                  : alert.severity === 'warning'
                                    ? 'bg-yellow-100 text-yellow-700 border border-yellow-200'
                                    : 'bg-blue-100 text-blue-700 border border-blue-200'
                              }`}>
                                <span className={`w-1.5 h-1.5 rounded-full ${
                                  alert.severity === 'critical' ? 'bg-red-500' :
                                  alert.severity === 'warning' ? 'bg-yellow-500' : 'bg-blue-500'
                                }`} />
                                {alert.severity === 'critical' ? '严重' : alert.severity === 'warning' ? '警告' : '信息'}
                              </span>
                            ) : (
                              <span className="text-xs text-gray-400">-</span>
                            )}
                          </td>
                          {/* Alert content/name */}
                          <td className="px-4 py-3 max-w-xs">
                            <p className="text-sm text-gray-700 truncate" title={alert.name}>
                              {alert.name || '未命名告警'}
                            </p>
                          </td>
                          {/* Alert target */}
                          <td className="px-4 py-3">
                            <span className="text-xs text-gray-500 font-mono bg-gray-50 px-1.5 py-0.5 rounded">
                              {alert.target || '-'}
                            </span>
                          </td>
                          {/* Timestamp */}
                          <td className="px-4 py-3 whitespace-nowrap">
                            {alert.timestamp ? (
                              <span className="text-xs text-gray-400">
                                {new Date(alert.timestamp).toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                              </span>
                            ) : (
                              <span className="text-xs text-gray-400">-</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="text-center py-12">
                  <div className="w-16 h-16 bg-emerald-50 rounded-2xl flex items-center justify-center mx-auto mb-4">
                    <CheckCircle2 className="w-8 h-8 text-emerald-400" />
                  </div>
                  <p className="text-sm font-medium text-gray-600 mb-1">一切正常</p>
                  <p className="text-xs text-gray-400">暂无告警信息，系统运行平稳</p>
                </div>
              )}
            </div>
          </div>

          {/* --- Platform Resources (2/5) --- */}
          <div className="lg:col-span-2 bg-white rounded-2xl border border-gray-100 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-50 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Cloud className="w-4.5 h-4.5 text-blue-500" />
                <div>
                  <h3 className="text-base font-semibold text-gray-800">平台资源</h3>
                  <p className="text-xs text-gray-400 mt-0.5">各云平台资源分布</p>
                </div>
              </div>
              <button
                onClick={() => setActivePage('cloud-platforms')}
                className="text-xs font-medium transition-colors flex items-center gap-1"
                style={{ color: '#513CC8' }}
              >
                管理 <ArrowRight className="w-3 h-3" />
              </button>
            </div>
            <div className="p-5">
              {loading ? (
                <div className="space-y-3">
                  {[1,2].map(i => (
                    <div key={i} className="h-20 bg-gray-50 rounded-xl animate-pulse" />
                  ))}
                </div>
              ) : (data?.platform_resources && data.platform_resources.length > 0) ? (
                <div className="space-y-3 max-h-80 overflow-y-auto" style={{ scrollbarWidth: 'thin' }}>
                  {data.platform_resources.map((p, i) => {
                    const badge = platformStatusBadge(p.status);
                    const BadgeIcon = badge.icon;
                    return (
                      <div
                        key={p.id || i}
                        className="p-4 rounded-xl border border-gray-100 hover:border-gray-200 transition-all hover:shadow-sm"
                      >
                        <div className="flex items-center justify-between mb-3">
                          <div className="flex items-center gap-2">
                            <div className="w-8 h-8 rounded-lg flex items-center justify-center bg-blue-50 ring-1 ring-blue-100">
                              <Cloud className="w-4 h-4 text-blue-500" />
                            </div>
                            <div>
                              <p className="text-sm font-semibold text-gray-700">{p.name}</p>
                              <span className="text-xs text-gray-400 uppercase">{p.type}</span>
                            </div>
                          </div>
                          <span className={`flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border ${badge.bg} ${badge.color} ${badge.border}`}>
                            <BadgeIcon className="w-3 h-3" />
                            {badge.label}
                          </span>
                        </div>
                        <div className="grid grid-cols-2 gap-3">
                          <div className="flex items-center gap-2 bg-gray-50 rounded-lg px-3 py-2">
                            <Server className="w-3.5 h-3.5 text-violet-500" />
                            <div>
                              <p className="text-lg font-bold text-gray-800 leading-none">{p.vm_count}</p>
                              <p className="text-xs text-gray-400">虚拟机</p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2 bg-gray-50 rounded-lg px-3 py-2">
                            <HardDrive className="w-3.5 h-3.5 text-emerald-500" />
                            <div>
                              <p className="text-lg font-bold text-gray-800 leading-none">{p.volume_count}</p>
                              <p className="text-xs text-gray-400">云硬盘</p>
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="text-center py-10">
                  <div className="w-14 h-14 bg-blue-50 rounded-2xl flex items-center justify-center mx-auto mb-3">
                    <Cloud className="w-7 h-7 text-blue-300" />
                  </div>
                  <p className="text-sm font-medium text-gray-600 mb-1">暂无云平台</p>
                  <p className="text-xs text-gray-400 mb-4">请先接入平台以查看资源数据</p>
                  <button
                    onClick={() => setActivePage('cloud-platforms')}
                    className="text-sm font-medium px-4 py-2 rounded-lg text-white transition-colors"
                    style={{ background: '#513CC8' }}
                  >
                    接入平台
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* ===== Component Health Section ===== */}
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm">
          <div className="px-6 py-4 border-b border-gray-50 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Activity className="w-4.5 h-4.5 text-[#513CC8]" />
              <div>
                <h3 className="text-base font-semibold text-gray-800">云平台组件健康状态</h3>
                <p className="text-xs text-gray-400 mt-0.5">OpenStack / EasyStack 核心服务运行情况</p>
              </div>
              {/* Ring health indicator */}
              {healthScore && (
                <div className="ml-4">
                  <RingProgress
                    value={healthScore.healthy}
                    max={healthScore.total}
                    size={48}
                    strokeWidth={5}
                    color={healthScore.pct === 100 ? '#10b981' : healthScore.pct >= 50 ? '#eab308' : '#ef4444'}
                  />
                </div>
              )}
            </div>
            {!loading && data?.components && (
              <span className={`flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border ${
                data.components.every(c => c.status === 'healthy')
                  ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                  : data.components.some(c => c.status === 'down')
                    ? 'bg-red-50 text-red-700 border-red-200'
                    : 'bg-yellow-50 text-yellow-700 border-yellow-200'
              }`}>
                <span className={`w-1.5 h-1.5 rounded-full ${
                  data.components.every(c => c.status === 'healthy')
                    ? 'bg-emerald-500 animate-pulse'
                    : data.components.some(c => c.status === 'down')
                      ? 'bg-red-500 animate-pulse'
                      : 'bg-yellow-500 animate-pulse'
                }`} />
                {data.components.every(c => c.status === 'healthy')
                  ? '全部正常'
                  : data.components.some(c => c.status === 'down')
                    ? '存在故障'
                    : '部分降级'}
              </span>
            )}
          </div>
          <div className="p-6">
            {loading ? (
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                {[1,2,3,4,5,6].map(i => (
                  <div key={i} className="h-24 bg-gray-50 rounded-xl animate-pulse" />
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                {(data?.components || []).map((comp, i) => {
                  const sc = statusColor(comp.status);
                  const CompIcon = componentIconMap[comp.name] || Activity;
                  return (
                    <div
                      key={i}
                      className={`relative p-4 rounded-xl border ${sc.border} ${sc.bg} transition-all hover:shadow-md group`}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <div className="w-9 h-9 rounded-lg bg-white flex items-center justify-center shadow-sm group-hover:scale-110 transition-transform">
                          <CompIcon className="w-4.5 h-4.5 text-gray-600" />
                        </div>
                        <div className="flex items-center gap-1">
                          <span className={`w-2 h-2 rounded-full ${sc.dot} ${comp.status === 'healthy' ? 'animate-pulse' : ''}`} />
                          <span className={`text-xs font-medium ${sc.text}`}>{sc.label}</span>
                        </div>
                      </div>
                      <p className="text-sm font-semibold text-gray-700 leading-snug">{comp.name}</p>
                      <p className="text-xs text-gray-400 mt-1 leading-relaxed">{comp.detail}</p>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* ===== Resource Distribution Overview ===== */}
        {!loading && data && (
          <div className="bg-white rounded-2xl border border-gray-100 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-50 flex items-center gap-2">
              <BarChart3 className="w-4.5 h-4.5 text-[#513CC8]" />
              <div>
                <h3 className="text-base font-semibold text-gray-800">资源分布概览</h3>
                <p className="text-xs text-gray-400 mt-0.5">各云平台资源占比与AI服务状态</p>
              </div>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {/* VM Distribution */}
                <div className="flex flex-col items-center">
                  <RingProgress
                    value={data.total_vms || 0}
                    max={Math.max(data.total_vms || 0, 10)}
                    size={80}
                    strokeWidth={8}
                    color="#7c3aed"
                    label="虚拟机"
                  />
                  <p className="text-sm text-gray-500 mt-3">虚拟机总数</p>
                  {data.platform_resources?.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-2 justify-center">
                      {data.platform_resources.map((p, idx) => (
                        <span key={idx} className="text-xs bg-violet-50 text-violet-600 px-2 py-0.5 rounded-full border border-violet-100">
                          {p.name}: {p.vm_count}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                {/* Volume Distribution */}
                <div className="flex flex-col items-center">
                  <RingProgress
                    value={data.total_volumes || 0}
                    max={Math.max(data.total_volumes || 0, 10)}
                    size={80}
                    strokeWidth={8}
                    color="#10b981"
                    label="云硬盘"
                  />
                  <p className="text-sm text-gray-500 mt-3">云硬盘总数</p>
                  {data.platform_resources?.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-2 justify-center">
                      {data.platform_resources.map((p, idx) => (
                        <span key={idx} className="text-xs bg-emerald-50 text-emerald-600 px-2 py-0.5 rounded-full border border-emerald-100">
                          {p.name}: {p.volume_count}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                {/* AI Services */}
                <div className="flex flex-col items-center">
                  <RingProgress
                    value={data.ai_providers || 0}
                    max={Math.max((data.ai_providers || 0) + (data.agents || 0), 10)}
                    size={80}
                    strokeWidth={8}
                    color="#f59e0b"
                    label="AI服务"
                  />
                  <p className="text-sm text-gray-500 mt-3">AI 服务状态</p>
                  <div className="mt-2 flex flex-wrap gap-2 justify-center">
                    <span className="text-xs bg-amber-50 text-amber-600 px-2 py-0.5 rounded-full border border-amber-100">
                      AI 模型: {data.ai_providers || 0}
                    </span>
                    <span className="text-xs bg-orange-50 text-orange-600 px-2 py-0.5 rounded-full border border-orange-100">
                      智能体: {data.agents || 0}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* ===== Cross-module Quick Links ===== */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[
            {
              title: '接入平台',
              desc: '管理 EasyStack、ZStack 等多云接入',
              icon: Cloud,
              page: 'cloud-platforms',
              gradient: 'from-blue-500 to-blue-600',
            },
            {
              title: 'AI 对话诊断',
              desc: '与智能体对话，快速排查云资源问题',
              icon: Bot,
              page: 'chat',
              gradient: 'from-violet-500 to-purple-600',
            },
            {
              title: '模型配置',
              desc: '配置 AI 模型参数以支持智能运维',
              icon: Cpu,
              page: 'ai-models',
              gradient: 'from-amber-500 to-orange-600',
            },
            {
              title: '定时任务',
              desc: '管理周期性巡检和自动化运维任务',
              icon: Clock,
              page: 'scheduled-tasks',
              gradient: 'from-teal-500 to-cyan-600',
            },
          ].map((link, i) => {
            const Icon = link.icon;
            return (
              <button
                key={i}
                onClick={() => setActivePage(link.page)}
                className="w-full flex items-center gap-4 p-4 bg-white rounded-2xl border border-gray-100 shadow-sm hover:shadow-md hover:border-[#513CC8] transition-all text-left group"
              >
                <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${link.gradient} flex items-center justify-center flex-shrink-0 group-hover:scale-110 transition-transform shadow-sm`}>
                  <Icon className="w-5 h-5 text-white" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-700 group-hover:text-[#513CC8] transition-colors">{link.title}</p>
                  <p className="text-xs text-gray-400 mt-0.5">{link.desc}</p>
                </div>
                <ArrowRight className="w-4 h-4 text-gray-300 group-hover:text-[#513CC8] transition-colors flex-shrink-0" />
              </button>
            );
          })}
        </div>

      </div>
    </div>
  );
}
