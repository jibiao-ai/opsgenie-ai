import React, { useEffect, useState, useCallback } from 'react';
import {
  FileText,
  Search,
  Filter,
  ChevronLeft,
  ChevronRight,
  User,
  Cloud,
  Bot,
  Plus,
  Edit2,
  Trash2,
  RefreshCw,
  Loader2,
  Clock,
  Monitor,
} from 'lucide-react';
import { getOperationLogs } from '../services/api';

// Module display config
const MODULE_CONFIG = {
  user:           { label: '用户管理', icon: User,  color: 'bg-blue-100 text-blue-700' },
  cloud_platform: { label: '云平台',   icon: Cloud, color: 'bg-green-100 text-green-700' },
  agent:          { label: '智能体',   icon: Bot,   color: 'bg-purple-100 text-purple-700' },
};

const ACTION_CONFIG = {
  create: { label: '新建', icon: Plus,   color: 'bg-emerald-100 text-emerald-700' },
  update: { label: '更新', icon: Edit2,  color: 'bg-amber-100 text-amber-700' },
  delete: { label: '删除', icon: Trash2, color: 'bg-red-100 text-red-700' },
};

function ModuleBadge({ module }) {
  const cfg = MODULE_CONFIG[module] || { label: module, icon: Monitor, color: 'bg-gray-100 text-gray-600' };
  const Icon = cfg.icon;
  return (
    <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium ${cfg.color}`}>
      <Icon className="w-3 h-3" />
      {cfg.label}
    </span>
  );
}

function ActionBadge({ action }) {
  const cfg = ACTION_CONFIG[action] || { label: action, icon: Edit2, color: 'bg-gray-100 text-gray-600' };
  const Icon = cfg.icon;
  return (
    <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium ${cfg.color}`}>
      <Icon className="w-3 h-3" />
      {cfg.label}
    </span>
  );
}

export default function OperationLogPage() {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [moduleFilter, setModuleFilter] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  const loadLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params = { page, page_size: pageSize };
      if (moduleFilter) params.module = moduleFilter;
      if (actionFilter) params.action = actionFilter;
      const res = await getOperationLogs(params);
      if (res.code === 0 && res.data) {
        setLogs(res.data.items || []);
        setTotal(res.data.total || 0);
      }
    } catch (err) {
      console.error('Failed to load operation logs:', err);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, moduleFilter, actionFilter]);

  useEffect(() => { loadLogs(); }, [loadLogs]);

  // Reset to page 1 when filters change
  useEffect(() => { setPage(1); }, [moduleFilter, actionFilter]);

  const totalPages = Math.ceil(total / pageSize) || 1;

  // Local search filter (search by username, target_name, detail)
  const filteredLogs = searchQuery
    ? logs.filter(
        (l) =>
          (l.username || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
          (l.target_name || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
          (l.detail || '').toLowerCase().includes(searchQuery.toLowerCase())
      )
    : logs;

  const formatTime = (ts) => {
    if (!ts) return '-';
    const d = new Date(ts);
    return d.toLocaleString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-6 space-y-6 max-w-6xl">
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">

          {/* Header: filters + search + refresh */}
          <div className="px-6 py-4 border-b border-gray-100 flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <h2 className="text-base font-semibold text-gray-800">操作日志</h2>
              <span className="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded-full">
                共 {total} 条
              </span>
            </div>
            <div className="flex items-center gap-3 flex-wrap">
              {/* Module filter */}
              <div className="flex items-center gap-1.5">
                <Filter className="w-4 h-4 text-gray-400" />
                <select
                  value={moduleFilter}
                  onChange={(e) => setModuleFilter(e.target.value)}
                  className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none bg-white"
                >
                  <option value="">全部模块</option>
                  <option value="user">用户管理</option>
                  <option value="cloud_platform">云平台</option>
                  <option value="agent">智能体</option>
                </select>
              </div>

              {/* Action filter */}
              <select
                value={actionFilter}
                onChange={(e) => setActionFilter(e.target.value)}
                className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none bg-white"
              >
                <option value="">全部操作</option>
                <option value="create">新建</option>
                <option value="update">更新</option>
                <option value="delete">删除</option>
              </select>

              {/* Search */}
              <div className="relative">
                <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="搜索用户名/目标..."
                  className="pl-9 pr-4 py-1.5 border border-gray-200 rounded-lg text-sm focus:ring-2 outline-none w-48"
                />
              </div>

              {/* Refresh */}
              <button
                onClick={loadLogs}
                disabled={loading}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm border border-gray-200 rounded-lg text-gray-600 hover:bg-gray-50 transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
                刷新
              </button>
            </div>
          </div>

          {/* Table */}
          {loading ? (
            <div className="flex items-center justify-center h-48">
              <Loader2 className="w-6 h-6 animate-spin" style={{ color: '#513CC8' }} />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="bg-gray-50 border-b border-gray-100">
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">时间</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">操作人</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">模块</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">操作</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">目标</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">详情</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">IP</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {filteredLogs.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="text-center py-16 text-gray-400">
                        <FileText className="w-10 h-10 mx-auto mb-2 opacity-30" />
                        <p className="text-sm">暂无操作日志</p>
                        <p className="text-xs mt-1 text-gray-300">
                          新建用户、接入云平台、管理智能体等操作将记录在此
                        </p>
                      </td>
                    </tr>
                  ) : (
                    filteredLogs.map((log) => (
                      <tr key={log.id} className="hover:bg-gray-50 transition-colors">
                        {/* Time */}
                        <td className="px-6 py-3 text-sm text-gray-500 whitespace-nowrap">
                          <div className="flex items-center gap-1.5">
                            <Clock className="w-3.5 h-3.5 text-gray-300" />
                            {formatTime(log.created_at)}
                          </div>
                        </td>
                        {/* User */}
                        <td className="px-6 py-3">
                          <div className="flex items-center gap-2">
                            <div
                              className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold text-white flex-shrink-0"
                              style={{ background: '#513CC8' }}
                            >
                              {(log.username || 'U').slice(0, 1).toUpperCase()}
                            </div>
                            <span className="text-sm text-gray-700">{log.username || '-'}</span>
                          </div>
                        </td>
                        {/* Module */}
                        <td className="px-6 py-3">
                          <ModuleBadge module={log.module} />
                        </td>
                        {/* Action */}
                        <td className="px-6 py-3">
                          <ActionBadge action={log.action} />
                        </td>
                        {/* Target */}
                        <td className="px-6 py-3 text-sm text-gray-700 max-w-[200px] truncate" title={log.target_name}>
                          {log.target_name || '-'}
                        </td>
                        {/* Detail */}
                        <td className="px-6 py-3 text-sm text-gray-500 max-w-[260px] truncate" title={log.detail}>
                          {log.detail || '-'}
                        </td>
                        {/* IP */}
                        <td className="px-6 py-3 text-sm text-gray-400 font-mono text-xs">
                          {log.ip || '-'}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}

          {/* Pagination */}
          {total > pageSize && (
            <div className="px-6 py-3 border-t border-gray-100 flex items-center justify-between">
              <span className="text-sm text-gray-400">
                第 {page} / {totalPages} 页，共 {total} 条
              </span>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="p-1.5 border border-gray-200 rounded-lg text-gray-500 hover:bg-gray-50 disabled:opacity-30 disabled:cursor-not-allowed transition"
                >
                  <ChevronLeft className="w-4 h-4" />
                </button>
                {/* Page numbers */}
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  let p;
                  if (totalPages <= 5) {
                    p = i + 1;
                  } else if (page <= 3) {
                    p = i + 1;
                  } else if (page >= totalPages - 2) {
                    p = totalPages - 4 + i;
                  } else {
                    p = page - 2 + i;
                  }
                  return (
                    <button
                      key={p}
                      onClick={() => setPage(p)}
                      className={`w-8 h-8 rounded-lg text-sm font-medium transition ${
                        p === page
                          ? 'text-white'
                          : 'text-gray-600 hover:bg-gray-100'
                      }`}
                      style={p === page ? { background: '#513CC8' } : {}}
                    >
                      {p}
                    </button>
                  );
                })}
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="p-1.5 border border-gray-200 rounded-lg text-gray-500 hover:bg-gray-50 disabled:opacity-30 disabled:cursor-not-allowed transition"
                >
                  <ChevronRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
