import React, { useEffect, useState } from 'react';
import { Workflow, Plus, Play, Trash2, Loader2 } from 'lucide-react';
import { getWorkflows, createWorkflow } from '../services/api';
import toast from 'react-hot-toast';

export default function WorkflowsPage() {
  const [workflows, setWorkflows] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const res = await getWorkflows();
        if (res.code === 0) setWorkflows(res.data || []);
      } catch (err) { console.error(err); }
      finally { setLoading(false); }
    })();
  }, []);

  // Demo workflows for display
  const demoWorkflows = [
    {
      id: 'demo-1', name: '云主机健康巡检', description: '定期检查所有云主机的运行状态、CPU/内存使用率、磁盘空间，生成巡检报告',
      steps: '1. 列举所有云主机 -> 2. 查询各主机CPU/内存指标 -> 3. 检查磁盘使用率 -> 4. 汇总生成报告', is_active: true,
    },
    {
      id: 'demo-2', name: '告警自动处理', description: '监听告警信息，根据告警类型自动执行修复操作',
      steps: '1. 查询活跃告警 -> 2. 分类告警类型 -> 3. 匹配处理策略 -> 4. 执行修复 -> 5. 发送通知', is_active: true,
    },
    {
      id: 'demo-3', name: '资源自动扩缩容', description: '根据负载情况自动调整云主机规格或数量',
      steps: '1. 监控资源使用 -> 2. 判断是否超阈值 -> 3. 计算目标规格 -> 4. 执行扩/缩容 -> 5. 验证结果', is_active: false,
    },
  ];

  const allWorkflows = [...workflows, ...demoWorkflows];

  return (
    <div className="h-full overflow-y-auto">
      <div className="p-4 sm:p-6 space-y-4 sm:space-y-6 w-full">
        {/* 页面头部卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
          <div className="px-6 py-4 flex items-center justify-between">
            <div>
              <h1 className="text-lg font-semibold text-gray-800">工作流</h1>
              <p className="text-sm text-gray-400 mt-0.5">自动化运维工作流编排与管理</p>
            </div>
            <button className="flex items-center gap-2 bg-[#513CC8] hover:bg-[#4230A6] text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors">
              <Plus className="w-4 h-4" /> 新建工作流
            </button>
          </div>
        </div>

        {/* 工作流列表卡片 */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6">
          {loading ? (
            <div className="flex items-center justify-center h-40">
              <Loader2 className="w-6 h-6 animate-spin text-[#513CC8]" />
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {allWorkflows.map((wf) => (
                <div key={wf.id} className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow">
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      <div className="w-10 h-10 bg-[#EEE9FB] rounded-lg flex items-center justify-center">
                        <Workflow className="w-5 h-5 text-[#513CC8]" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-gray-800">{wf.name}</h3>
                          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${wf.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                            {wf.is_active ? '已启用' : '已停用'}
                          </span>
                        </div>
                        <p className="text-sm text-gray-500 mt-1">{wf.description}</p>
                        <p className="text-xs text-gray-400 mt-2">{wf.steps}</p>
                      </div>
                    </div>
                    <div className="flex gap-1">
                      <button className="p-1.5 text-gray-400 hover:text-green-500 hover:bg-green-50 rounded-lg transition-colors">
                        <Play className="w-4 h-4" />
                      </button>
                      <button className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
