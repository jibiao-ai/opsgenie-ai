import React, { useEffect, useState } from 'react';
import {
  MessageSquare,
  Bot,
  Zap,
  Cloud,
  Cpu,
  TrendingUp,
  TrendingDown,
  ArrowRight,
  Plus,
  Settings,
  FileText,
  Activity,
} from 'lucide-react';
import { getDashboard } from '../services/api';
import useStore from '../store/useStore';

export default function DashboardPage() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const setActivePage = useStore((s) => s.setActivePage);

  useEffect(() => {
    loadDashboard();
  }, []);

  const loadDashboard = async () => {
    try {
      const res = await getDashboard();
      if (res.code === 0) {
        setStats(res.data);
      }
    } catch (err) {
      console.error('Failed to load dashboard:', err);
    } finally {
      setLoading(false);
    }
  };

  // 顶部 4 个统计卡片
  const statCards = [
    {
      label: '云平台数量',
      value: stats?.cloud_platforms ?? '--',
      icon: Cloud,
      iconBg: 'bg-blue-100',
      iconColor: 'text-blue-600',
      trend: '已接入',
      trendUp: true,
    },
    {
      label: 'AI 模型数',
      value: stats?.ai_models ?? stats?.agents ?? '--',
      icon: Cpu,
      iconBg: 'bg-purple-100',
      iconColor: 'text-purple-600',
      trend: '已配置',
      trendUp: true,
    },
    {
      label: '活跃 Agent 数',
      value: stats?.agents ?? '--',
      icon: Bot,
      iconBg: 'bg-emerald-100',
      iconColor: 'text-emerald-600',
      trend: '运行中',
      trendUp: true,
    },
    {
      label: '今日对话数',
      value: stats?.conversations ?? '--',
      icon: MessageSquare,
      iconBg: 'bg-orange-100',
      iconColor: 'text-orange-600',
      trend: '较昨日',
      trendUp: true,
    },
  ];

  // 快捷操作
  const quickActions = [
    {
      label: '新建对话',
      desc: '与 AI 智能体开始新会话',
      icon: MessageSquare,
      iconBg: 'bg-[#EEE9FB]',
      iconColor: 'text-[#513CC8]',
      page: 'chat',
    },
    {
      label: '配置模型',
      desc: '添加或更新 AI 模型参数',
      icon: Cpu,
      iconBg: 'bg-blue-50',
      iconColor: 'text-blue-600',
      page: 'ai-models',
    },
    {
      label: '接入云平台',
      desc: '添加 EasyStack / ZStack',
      icon: Cloud,
      iconBg: 'bg-emerald-50',
      iconColor: 'text-emerald-600',
      page: 'cloud-platforms',
    },
    {
      label: '查看技能',
      desc: '浏览平台可用技能列表',
      icon: Zap,
      iconBg: 'bg-amber-50',
      iconColor: 'text-amber-600',
      page: 'skills',
    },
  ];

  // 模拟最近对话（实际可从 stats.recent_conversations 取）
  const recentConvs = stats?.recent_conversations || [];

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-6 space-y-6 max-w-7xl">

        {/* === 顶部 4 个统计卡片 === */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {statCards.map((card, i) => {
            const Icon = card.icon;
            return (
              <div
                key={i}
                className="bg-white rounded-xl border border-gray-200 shadow-sm p-5 flex flex-col gap-3 hover:shadow-md transition-shadow"
              >
                <div className="flex items-start justify-between">
                  <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${card.iconBg}`}>
                    <Icon className={`w-5 h-5 ${card.iconColor}`} />
                  </div>
                  {loading ? (
                    <div className="w-8 h-4 bg-gray-100 rounded animate-pulse" />
                  ) : (
                    <span className="flex items-center gap-0.5 text-xs text-emerald-600">
                      <TrendingUp className="w-3 h-3" />
                      {card.trend}
                    </span>
                  )}
                </div>
                {loading ? (
                  <>
                    <div className="h-8 w-16 bg-gray-100 rounded animate-pulse" />
                    <div className="h-4 w-24 bg-gray-50 rounded animate-pulse" />
                  </>
                ) : (
                  <>
                    <p className="text-3xl font-bold text-gray-800">{card.value}</p>
                    <p className="text-sm text-gray-500">{card.label}</p>
                  </>
                )}
              </div>
            );
          })}
        </div>

        {/* === 中间区域：最近对话 + 快捷操作 === */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

          {/* 最近对话列表（占 2/3） */}
          <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm">
            {/* 卡片头 */}
            <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
              <div>
                <h2 className="text-base font-semibold text-gray-800">最近对话</h2>
                <p className="text-sm text-gray-400 mt-0.5">近期 AI 对话记录</p>
              </div>
              <button
                onClick={() => setActivePage('chat')}
                className="flex items-center gap-1 text-sm font-medium transition-colors"
                style={{ color: '#513CC8' }}
              >
                查看全部 <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
            {/* 卡片体 */}
            <div className="p-6">
              {loading ? (
                <div className="space-y-3">
                  {[1, 2, 3].map((i) => (
                    <div key={i} className="flex gap-3 animate-pulse">
                      <div className="w-8 h-8 bg-gray-100 rounded-full flex-shrink-0" />
                      <div className="flex-1 space-y-1.5">
                        <div className="h-4 bg-gray-100 rounded w-1/2" />
                        <div className="h-3 bg-gray-50 rounded w-3/4" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : recentConvs.length > 0 ? (
                <div className="space-y-1">
                  {recentConvs.slice(0, 6).map((conv, i) => (
                    <div
                      key={conv.id || i}
                      className="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors group"
                      onClick={() => setActivePage('chat')}
                    >
                      <div className="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 bg-[#EEE9FB]">
                        <MessageSquare className="w-4 h-4 text-[#513CC8]" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-700 truncate group-hover:text-[#513CC8] transition-colors">
                          {conv.title || conv.name || `对话 ${i + 1}`}
                        </p>
                        <p className="text-xs text-gray-400 truncate">
                          {conv.updated_at
                            ? new Date(conv.updated_at).toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
                            : '刚刚'}
                        </p>
                      </div>
                      <ArrowRight className="w-3.5 h-3.5 text-gray-300 group-hover:text-[#513CC8] transition-colors flex-shrink-0" />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-10">
                  <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <MessageSquare className="w-6 h-6 text-gray-300" />
                  </div>
                  <p className="text-sm text-gray-400 mb-3">暂无对话记录</p>
                  <button
                    onClick={() => setActivePage('chat')}
                    className="text-sm font-medium px-4 py-2 rounded-lg text-white transition-colors"
                    style={{ background: '#513CC8' }}
                  >
                    <span className="flex items-center gap-1.5">
                      <Plus className="w-4 h-4" /> 开始第一个对话
                    </span>
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* 快捷操作（占 1/3） */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
            <div className="px-6 py-4 border-b border-gray-100">
              <h2 className="text-base font-semibold text-gray-800">快捷操作</h2>
              <p className="text-sm text-gray-400 mt-0.5">常用功能入口</p>
            </div>
            <div className="p-4 space-y-2">
              {quickActions.map((action, i) => {
                const Icon = action.icon;
                return (
                  <button
                    key={i}
                    onClick={() => setActivePage(action.page)}
                    className="w-full flex items-center gap-3 px-4 py-3 rounded-xl border border-gray-100 hover:border-[#513CC8] hover:bg-[#EEE9FB] transition-all text-left group"
                  >
                    <div className={`w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 ${action.iconBg} group-hover:scale-110 transition-transform`}>
                      <Icon className={`w-4 h-4 ${action.iconColor}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-700 group-hover:text-[#513CC8] transition-colors">
                        {action.label}
                      </p>
                      <p className="text-xs text-gray-400 truncate">{action.desc}</p>
                    </div>
                    <ArrowRight className="w-3.5 h-3.5 text-gray-300 group-hover:text-[#513CC8] transition-colors flex-shrink-0" />
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* === 平台状态概览 === */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
            <div>
              <h2 className="text-base font-semibold text-gray-800">平台状态</h2>
              <p className="text-sm text-gray-400 mt-0.5">各模块运行状态一览</p>
            </div>
            <span className="flex items-center gap-1.5 text-xs text-emerald-600 bg-emerald-50 px-2 py-1 rounded-full border border-emerald-200">
              <span className="w-1.5 h-1.5 bg-emerald-500 rounded-full animate-pulse" />
              系统正常
            </span>
          </div>
          <div className="p-6 grid grid-cols-2 md:grid-cols-4 gap-4">
            {[
              { label: 'AI 服务', desc: '模型接口正常', icon: Cpu, ok: true },
              { label: '对话服务', desc: '实时消息正常', icon: MessageSquare, ok: true },
              { label: '智能体', desc: `共 ${stats?.agents ?? 0} 个`, icon: Bot, ok: true },
              { label: '任务调度', desc: '定时任务运行中', icon: Activity, ok: true },
            ].map((item, i) => {
              const Icon = item.icon;
              return (
                <div key={i} className="flex items-center gap-3 p-3 bg-gray-50 rounded-xl">
                  <div className="w-8 h-8 bg-white rounded-lg flex items-center justify-center shadow-sm flex-shrink-0">
                    <Icon className="w-4 h-4 text-gray-500" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-gray-700">{item.label}</p>
                    <p className="text-xs text-gray-400 truncate">{item.desc}</p>
                  </div>
                  <span className={`w-2 h-2 rounded-full flex-shrink-0 ${item.ok ? 'bg-emerald-500' : 'bg-red-500'}`} />
                </div>
              );
            })}
          </div>
        </div>

      </div>
    </div>
  );
}
