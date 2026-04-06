import React, { useEffect, useState, useCallback } from 'react';
import { Clock, Plus, Play, Pause, Trash2, Loader2, X, AlertTriangle, CheckCircle } from 'lucide-react';
import { getScheduledTasks, createScheduledTask, updateScheduledTask, deleteScheduledTask } from '../services/api';

const TASK_TYPE_OPTIONS = [
  { value: 'health_check', label: '健康巡检' },
  { value: 'alert_summary', label: '告警汇总' },
  { value: 'resource_analysis', label: '资源分析' },
  { value: 'backup', label: '数据备份' },
  { value: 'cleanup', label: '资源清理' },
  { value: 'custom', label: '自定义' },
];

const CRON_PRESETS = [
  { label: '每小时', value: '0 * * * *' },
  { label: '每6小时', value: '0 */6 * * *' },
  { label: '每天 08:00', value: '0 8 * * *' },
  { label: '每天 00:00', value: '0 0 * * *' },
  { label: '每周一 08:00', value: '0 8 * * 1' },
  { label: '每月1日 00:00', value: '0 0 1 * *' },
];

function getTaskTypeLabel(type) {
  const opt = TASK_TYPE_OPTIONS.find(o => o.value === type);
  return opt ? opt.label : type;
}

export default function ScheduledTasksPage() {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [creating, setCreating] = useState(false);
  const [togglingId, setTogglingId] = useState(null);
  const [deletingId, setDeletingId] = useState(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState(null);
  const [toast, setToast] = useState(null);

  // Form state
  const [form, setForm] = useState({
    name: '',
    cron_expr: '0 8 * * *',
    task_type: 'health_check',
    config: '',
  });
  const [formErrors, setFormErrors] = useState({});

  const showToast = useCallback((msg, type = 'success') => {
    setToast({ msg, type });
    setTimeout(() => setToast(null), 3000);
  }, []);

  const fetchTasks = useCallback(async () => {
    try {
      const res = await getScheduledTasks();
      if (res.code === 0) setTasks(res.data || []);
    } catch (err) {
      console.error(err);
      showToast('获取任务列表失败', 'error');
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => { fetchTasks(); }, [fetchTasks]);

  // --- Create ---
  const validateForm = () => {
    const errors = {};
    if (!form.name.trim()) errors.name = '请输入任务名称';
    if (!form.cron_expr.trim()) errors.cron_expr = '请输入 Cron 表达式';
    if (!form.task_type) errors.task_type = '请选择任务类型';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleCreate = async () => {
    if (!validateForm()) return;
    setCreating(true);
    try {
      const res = await createScheduledTask({
        name: form.name.trim(),
        cron_expr: form.cron_expr.trim(),
        task_type: form.task_type,
        config: form.config.trim() || '{}',
        is_active: true,
      });
      if (res.code === 0) {
        showToast('任务创建成功');
        setShowCreateModal(false);
        setForm({ name: '', cron_expr: '0 8 * * *', task_type: 'health_check', config: '' });
        setFormErrors({});
        fetchTasks();
      } else {
        showToast(res.message || '创建失败', 'error');
      }
    } catch (err) {
      showToast(err?.message || '创建失败', 'error');
    } finally {
      setCreating(false);
    }
  };

  // --- Toggle ---
  const handleToggle = async (task) => {
    setTogglingId(task.id);
    try {
      const res = await updateScheduledTask(task.id, { is_active: !task.is_active });
      if (res.code === 0) {
        showToast(task.is_active ? '任务已停止' : '任务已启用');
        fetchTasks();
      } else {
        showToast(res.message || '操作失败', 'error');
      }
    } catch (err) {
      showToast(err?.message || '操作失败', 'error');
    } finally {
      setTogglingId(null);
    }
  };

  // --- Delete ---
  const handleDelete = async (id) => {
    setDeletingId(id);
    try {
      const res = await deleteScheduledTask(id);
      if (res.code === 0) {
        showToast('任务已删除');
        setConfirmDeleteId(null);
        fetchTasks();
      } else {
        showToast(res.message || '删除失败', 'error');
      }
    } catch (err) {
      showToast(err?.message || '删除失败', 'error');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        {/* Toast */}
        {toast && (
          <div className={`fixed top-6 right-6 z-50 flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg text-sm font-medium transition-all ${
            toast.type === 'error' ? 'bg-red-50 text-red-700 border border-red-200' : 'bg-green-50 text-green-700 border border-green-200'
          }`}>
            {toast.type === 'error' ? <AlertTriangle className="w-4 h-4" /> : <CheckCircle className="w-4 h-4" />}
            {toast.msg}
          </div>
        )}

        {/* Header */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">定时任务</h1>
              <p className="text-sm text-gray-400 mt-0.5">配置定时执行的自动化运维任务</p>
            </div>
            <button
              onClick={() => setShowCreateModal(true)}
              className="flex items-center gap-2 bg-[#513CC8] hover:bg-[#4230A6] text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
            >
              <Plus className="w-4 h-4" /> 新建任务
            </button>
          </div>
        </div>

        {/* Task List */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          {loading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="w-6 h-6 animate-spin text-[#513CC8]" />
            </div>
          ) : tasks.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-40 text-gray-400">
              <Clock className="w-10 h-10 mb-2 opacity-40" />
              <p className="text-sm">暂无定时任务，点击上方按钮新建</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[600px]">
                <thead>
                  <tr className="bg-gray-50 border-b border-gray-100">
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">任务名称</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Cron 表达式</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">类型</th>
                    <th className="text-left px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">状态</th>
                    <th className="text-right px-6 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {tasks.map((task) => (
                    <tr key={task.id} className="hover:bg-gray-50 transition-colors">
                      <td className="px-6 py-3.5">
                        <div className="flex items-center gap-2">
                          <Clock className="w-4 h-4 text-[#513CC8]" />
                          <span className="text-sm font-medium text-gray-700">{task.name}</span>
                        </div>
                      </td>
                      <td className="px-6 py-3.5">
                        <code className="text-xs bg-gray-100 px-2 py-0.5 rounded text-gray-600">{task.cron_expr}</code>
                      </td>
                      <td className="px-6 py-3.5">
                        <span className="text-xs px-2 py-0.5 bg-purple-50 text-purple-600 rounded-full">
                          {getTaskTypeLabel(task.task_type)}
                        </span>
                      </td>
                      <td className="px-6 py-3.5">
                        <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                          task.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                        }`}>
                          {task.is_active ? '运行中' : '已停止'}
                        </span>
                      </td>
                      <td className="px-6 py-3.5 text-right">
                        <div className="flex justify-end gap-1">
                          {/* Toggle */}
                          <button
                            onClick={() => handleToggle(task)}
                            disabled={togglingId === task.id}
                            title={task.is_active ? '停止任务' : '启用任务'}
                            className="p-1.5 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors disabled:opacity-50"
                          >
                            {togglingId === task.id ? (
                              <Loader2 className="w-4 h-4 animate-spin" />
                            ) : task.is_active ? (
                              <Pause className="w-4 h-4" />
                            ) : (
                              <Play className="w-4 h-4" />
                            )}
                          </button>
                          {/* Delete */}
                          {confirmDeleteId === task.id ? (
                            <div className="flex items-center gap-1">
                              <button
                                onClick={() => handleDelete(task.id)}
                                disabled={deletingId === task.id}
                                className="px-2 py-1 text-xs bg-red-500 text-white rounded hover:bg-red-600 transition-colors disabled:opacity-50"
                              >
                                {deletingId === task.id ? '删除中...' : '确认'}
                              </button>
                              <button
                                onClick={() => setConfirmDeleteId(null)}
                                className="px-2 py-1 text-xs bg-gray-200 text-gray-600 rounded hover:bg-gray-300 transition-colors"
                              >
                                取消
                              </button>
                            </div>
                          ) : (
                            <button
                              onClick={() => setConfirmDeleteId(task.id)}
                              title="删除任务"
                              className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden">
            {/* Modal Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
              <h2 className="text-base font-semibold text-gray-800">新建定时任务</h2>
              <button onClick={() => { setShowCreateModal(false); setFormErrors({}); }} className="p-1 hover:bg-gray-100 rounded-lg transition-colors">
                <X className="w-5 h-5 text-gray-400" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="px-6 py-5 space-y-4">
              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">任务名称 <span className="text-red-400">*</span></label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  placeholder="例如：每日巡检"
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#513CC8]/30 focus:border-[#513CC8] ${
                    formErrors.name ? 'border-red-300' : 'border-gray-200'
                  }`}
                />
                {formErrors.name && <p className="mt-1 text-xs text-red-500">{formErrors.name}</p>}
              </div>

              {/* Cron Expr */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Cron 表达式 <span className="text-red-400">*</span></label>
                <input
                  type="text"
                  value={form.cron_expr}
                  onChange={e => setForm(f => ({ ...f, cron_expr: e.target.value }))}
                  placeholder="0 8 * * *"
                  className={`w-full px-3 py-2 border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-[#513CC8]/30 focus:border-[#513CC8] ${
                    formErrors.cron_expr ? 'border-red-300' : 'border-gray-200'
                  }`}
                />
                {formErrors.cron_expr && <p className="mt-1 text-xs text-red-500">{formErrors.cron_expr}</p>}
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {CRON_PRESETS.map(p => (
                    <button
                      key={p.value}
                      type="button"
                      onClick={() => setForm(f => ({ ...f, cron_expr: p.value }))}
                      className={`text-xs px-2 py-0.5 rounded-full border transition-colors ${
                        form.cron_expr === p.value
                          ? 'bg-[#513CC8] text-white border-[#513CC8]'
                          : 'bg-gray-50 text-gray-500 border-gray-200 hover:border-[#513CC8] hover:text-[#513CC8]'
                      }`}
                    >
                      {p.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Task Type */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">任务类型 <span className="text-red-400">*</span></label>
                <select
                  value={form.task_type}
                  onChange={e => setForm(f => ({ ...f, task_type: e.target.value }))}
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#513CC8]/30 focus:border-[#513CC8] ${
                    formErrors.task_type ? 'border-red-300' : 'border-gray-200'
                  }`}
                >
                  {TASK_TYPE_OPTIONS.map(o => (
                    <option key={o.value} value={o.value}>{o.label}</option>
                  ))}
                </select>
                {formErrors.task_type && <p className="mt-1 text-xs text-red-500">{formErrors.task_type}</p>}
              </div>

              {/* Config (optional) */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">配置 (JSON, 可选)</label>
                <textarea
                  value={form.config}
                  onChange={e => setForm(f => ({ ...f, config: e.target.value }))}
                  placeholder='{"target": "all_platforms"}'
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-[#513CC8]/30 focus:border-[#513CC8] resize-none"
                />
              </div>
            </div>

            {/* Modal Footer */}
            <div className="flex justify-end gap-3 px-6 py-4 border-t border-gray-100 bg-gray-50">
              <button
                onClick={() => { setShowCreateModal(false); setFormErrors({}); }}
                className="px-4 py-2 text-sm text-gray-600 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleCreate}
                disabled={creating}
                className="flex items-center gap-2 px-4 py-2 text-sm text-white bg-[#513CC8] rounded-lg hover:bg-[#4230A6] transition-colors disabled:opacity-50"
              >
                {creating && <Loader2 className="w-4 h-4 animate-spin" />}
                {creating ? '创建中...' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
