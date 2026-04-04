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
import useStore from '../store/useStore';

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
};

const THEMES = [
  { id: 'light', label: '白色', bg: '#ffffff', border: '#e5e7eb' },
  { id: 'dark',  label: '黑色', bg: '#0f0e17', border: '#374151' },
  { id: 'blue',  label: '蓝色', bg: '#0d1b4b', border: '#1e3a8a' },
];

export default function MainLayout() {
  const activePage = useStore((s) => s.activePage);
  const theme = useStore((s) => s.theme);
  const setTheme = useStore((s) => s.setTheme);
  const PageComponent = pageComponents[activePage] || ChatPage;

  // Apply theme on mount and whenever theme changes
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top Header Bar */}
        <header className="h-16 bg-primary flex items-center justify-end px-6 flex-shrink-0 shadow-md">
          {/* Theme Switcher */}
          <div className="flex items-center gap-2">
            <span className="text-white/70 text-xs mr-1">主题</span>
            {THEMES.map((t) => (
              <button
                key={t.id}
                title={t.label}
                onClick={() => setTheme(t.id)}
                className={`w-6 h-6 rounded-full border-2 transition-all ${
                  theme === t.id
                    ? 'border-white scale-110 shadow-lg'
                    : 'border-white/40 hover:border-white/80'
                }`}
                style={{ backgroundColor: t.bg, borderColor: theme === t.id ? '#ffffff' : t.border }}
              />
            ))}
          </div>
        </header>

        <main className="flex-1 overflow-hidden">
          <PageComponent />
        </main>
      </div>
    </div>
  );
}
