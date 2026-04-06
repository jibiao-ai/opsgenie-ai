import React, { useEffect } from 'react';
import Sidebar from './Sidebar';
import ChatPage from '../pages/ChatPage';
import DashboardPage from '../pages/DashboardPage';
import AgentsPage from '../pages/AgentsPage';
import SkillsPage from '../pages/SkillsPage';
import WorkflowsPage from '../pages/WorkflowsPage';
import ScheduledTasksPage from '../pages/ScheduledTasksPage';
import UsersPage from '../pages/UsersPage';
import AIModelsPage from '../pages/AIModelsPage';
import CloudPlatformPage from '../pages/CloudPlatformPage';
import ResourceMonitorPage from '../pages/ResourceMonitorPage';
import OperationLogPage from '../pages/OperationLogPage';
import useStore from '../store/useStore';
import { Bell, User } from 'lucide-react';

const pageComponents = {
  dashboard: DashboardPage,
  chat: ChatPage,
  agents: AgentsPage,
  skills: SkillsPage,
  workflows: WorkflowsPage,
  'scheduled-tasks': ScheduledTasksPage,
  users: UsersPage,
  'ai-models': AIModelsPage,
  'cloud-platforms': CloudPlatformPage,
  'resource-monitor': ResourceMonitorPage,
  'operation-logs': OperationLogPage,
};

// 页面元信息（标题 + 副标题）
const PAGE_META = {
  dashboard:         { title: '仪表盘',     subtitle: 'OpsGenie AI 智能运维平台概览' },
  chat:              { title: '即时对话',   subtitle: '与 AI 智能体实时交互，处理运维任务' },
  agents:            { title: '智能体',     subtitle: '管理和配置 AI 智能体' },
  skills:            { title: '技能商店',   subtitle: '查看和管理平台技能' },
  workflows:         { title: '工作流',     subtitle: '编排和管理自动化工作流' },
  'scheduled-tasks': { title: '定时任务',   subtitle: '管理周期性自动化任务' },
  users:             { title: '用户管理',   subtitle: '管理平台用户账号和权限' },
  'ai-models':       { title: '模型配置',   subtitle: '配置 AI 服务提供商参数' },
  'cloud-platforms': { title: '接入平台', subtitle: '管理 EasyStack、ZStack 等多云接入' },
  'resource-monitor': { title: '资源监控', subtitle: '实时监控云平台资源状态与告警信息' },
  'operation-logs':   { title: '操作日志', subtitle: '记录平台关键操作，包括用户管理、云平台接入、智能体管理等' },
};

const THEMES = [
  { id: 'light', label: '白色主题', bg: '#ffffff', border: '#e5e7eb' },
  { id: 'dark',  label: '暗色主题', bg: '#0f0e17', border: '#374151' },
  { id: 'blue',  label: '蓝色主题', bg: '#0d1b4b', border: '#1e3a8a' },
];

export default function MainLayout() {
  const activePage = useStore((s) => s.activePage);
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  const user = useStore((s) => s.user);
  const PageComponent = pageComponents[activePage] || ChatPage;
  const meta = PAGE_META[activePage] || { title: activePage, subtitle: '' };

  // Apply theme on mount and whenever theme changes
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        {/* 顶部 Header */}
        <header className="h-14 bg-white border-b border-gray-200 flex items-center px-6 flex-shrink-0 z-10">
          {/* 左侧：页面标题 + 副标题 */}
          <div className="flex-1 min-w-0">
            <h1 className="text-lg font-semibold text-gray-800 leading-tight">{meta.title}</h1>
            {meta.subtitle && (
              <p className="text-sm text-gray-400 leading-tight hidden sm:block">{meta.subtitle}</p>
            )}
          </div>

          {/* 右侧操作区 */}
          <div className="flex items-center gap-3 flex-shrink-0">
            {/* 主题切换 */}
            <div className="flex items-center gap-1.5 bg-gray-50 border border-gray-200 rounded-lg px-2 py-1.5">
              {THEMES.map((t) => (
                <button
                  key={t.id}
                  title={t.label}
                  onClick={() => setTheme(t.id)}
                  className={`w-4 h-4 rounded-full transition-all ring-offset-1 ${
                    theme === t.id
                      ? 'ring-2 ring-[#513CC8] scale-110'
                      : 'hover:ring-2 hover:ring-gray-300'
                  }`}
                  style={{ backgroundColor: t.bg, border: `1.5px solid ${t.border}` }}
                />
              ))}
            </div>

            {/* 通知铃铛 */}
            <button
              className="w-8 h-8 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors relative"
              title="通知"
            >
              <Bell className="w-4 h-4" />
              {/* 未读红点 */}
              <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-red-500 rounded-full" />
            </button>

            {/* 用户头像 */}
            <div className="flex items-center gap-2">
              <div
                className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white flex-shrink-0"
                style={{ background: '#513CC8' }}
              >
                {(user?.username || 'U').slice(0, 1).toUpperCase()}
              </div>
              <span className="text-sm font-medium text-gray-700 hidden md:block">
                {user?.username || 'admin'}
              </span>
            </div>
          </div>
        </header>

        {/* 内容区 */}
        <main className="flex-1 overflow-hidden bg-gray-50">
          <PageComponent />
        </main>
      </div>
    </div>
  );
}
