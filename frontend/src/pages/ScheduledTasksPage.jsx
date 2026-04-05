import React, { useEffect, useState } from 'react';
import { Clock, Plus, Play, Pause, Trash2, Loader2 } from 'lucide-react';
import { getScheduledTasks } from '../services/api';

export default function ScheduledTasksPage() {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const res = await getScheduledTasks();
        if (res.code === 0) setTasks(res.data || []);
      } catch (err) { console.error(err); }
      finally { setLoading(false); }
    })();
  }, []);

  const demoTasks = [
    { id: 'dt-1', name: '每日巡检', cron_expr: '0 8 * * *', task_type: 'health_check', is_active: true, last_run_at: null, next_run_at: null },
    { id: 'dt-2', name: '告警汇总报告', cron_expr: '0 */6 * * *', task_type: 'alert_summary', is_active: true, last_run_at: null, next_run_at: null },
    { id: 'dt-3', name: '资源使用分析', cron_expr: '0 0 * * 1', task_type: 'resource_analysis', is_active: false, last_run_at: null, next_run_at: null },
  ];

  const allTasks = [...tasks, ...demoTasks];

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-6 space-y-6 max-w-5xl">
        {/* 页面头部卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">定时任务</h1>
              <p className="text-sm text-gray-400 mt-0.5">配置定时执行的自动化运维任务</p>
            </div>
            <button className="flex items-center gap-2 bg-[#513CC8] hover:bg-[#4230A6] text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
              <Plus className="w-4 h-4" /> 新建任务
            </button>
          </div>
        </div>

        {/* 任务列表卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          {loading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="w-6 h-6 animate-spin text-[#513CC8]" />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
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
                  {allTasks.map((task) => (
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
                        <span className="text-xs px-2 py-0.5 bg-purple-50 text-purple-600 rounded-full">{task.task_type}</span>
                      </td>
                      <td className="px-6 py-3.5">
                        <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${task.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                          {task.is_active ? '运行中' : '已停止'}
                        </span>
                      </td>
                      <td className="px-6 py-3.5 text-right">
                        <div className="flex justify-end gap-1">
                          <button className="p-1.5 text-gray-400 hover:text-[#513CC8] hover:bg-[#EEE9FB] rounded-lg transition-colors">
                            {task.is_active ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                          </button>
                          <button className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors">
                            <Trash2 className="w-4 h-4" />
                          </button>
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
    </div>
  );
}
