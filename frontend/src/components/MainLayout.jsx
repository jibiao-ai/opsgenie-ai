import React from 'react';
import Sidebar from './Sidebar';
import ChatPage from '../pages/ChatPage';
import DashboardPage from '../pages/DashboardPage';
import AgentsPage from '../pages/AgentsPage';
import SkillsPage from '../pages/SkillsPage';
import WorkflowsPage from '../pages/WorkflowsPage';
import ScheduledTasksPage from '../pages/ScheduledTasksPage';
import UsersPage from '../pages/UsersPage';
import AIModelsPage from '../pages/AIModelsPage';
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
};

export default function MainLayout() {
  const activePage = useStore((s) => s.activePage);
  const PageComponent = pageComponents[activePage] || ChatPage;

  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <main className="flex-1 overflow-hidden">
        <PageComponent />
      </main>
    </div>
  );
}
