import React from 'react';
import {
  LayoutDashboard,
  MessageSquare,
  Bot,
  Zap,
  Workflow,
  Clock,
  Users,
  LogOut,
  User,
  Menu,
  Cpu,
  Cloud,
} from 'lucide-react';
import useStore from '../store/useStore';
import { useNavigate } from 'react-router-dom';

const menuItems = [
  { id: 'dashboard', label: '仪表盘', icon: LayoutDashboard },
  { id: 'chat', label: '即时对话', icon: MessageSquare },
  { id: 'agents', label: '智能体', icon: Bot },
  { id: 'skills', label: '技能中心', icon: Zap },
  { id: 'workflows', label: '工作流', icon: Workflow },
  { id: 'scheduled-tasks', label: '定时任务', icon: Clock },
  { id: 'ai-models', label: '模型配置', icon: Cpu },
  { id: 'cloud-platforms', label: '接入云平台', icon: Cloud },
];

const adminMenuItems = [
  { id: 'users', label: '用户', icon: Users },
];

export default function Sidebar() {
  const { activePage, setActivePage, user, logout, sidebarCollapsed, toggleSidebar } = useStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className={`flex flex-col bg-white border-r border-gray-200 transition-all duration-300 ${sidebarCollapsed ? 'w-16' : 'w-56'}`}>
      {/* Header */}
      <div className="flex items-center h-16 px-4 border-b border-gray-200 bg-primary">
        <button onClick={toggleSidebar} className="text-white hover:bg-primary-700 p-1 rounded">
          <Menu className="w-5 h-5" />
        </button>
        {!sidebarCollapsed && (
          <h1 className="ml-3 text-white font-semibold text-sm whitespace-nowrap">AIOPS智能运维平台</h1>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-2 overflow-y-auto">
        {menuItems.map((item) => {
          const Icon = item.icon;
          const isActive = activePage === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setActivePage(item.id)}
              className={`w-full flex items-center px-4 py-2.5 text-sm transition-colors ${
                isActive
                  ? 'bg-primary-50 text-primary border-r-2 border-primary'
                  : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
              }`}
              title={sidebarCollapsed ? item.label : undefined}
            >
              <Icon className={`w-5 h-5 ${isActive ? 'text-primary' : 'text-gray-400'}`} />
              {!sidebarCollapsed && (
                <span className="ml-3 whitespace-nowrap">{item.label}</span>
              )}
            </button>
          );
        })}

        {/* Admin section */}
        {user?.role === 'admin' && (
          <>
            {!sidebarCollapsed && (
              <div className="px-4 py-2 mt-2">
                <span className="text-xs font-medium text-gray-400 uppercase tracking-wider">
                  权限管理
                </span>
              </div>
            )}
            {adminMenuItems.map((item) => {
              const Icon = item.icon;
              const isActive = activePage === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => setActivePage(item.id)}
                  className={`w-full flex items-center px-4 py-2.5 text-sm transition-colors ${
                    isActive
                      ? 'bg-primary-50 text-primary border-r-2 border-primary'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                  title={sidebarCollapsed ? item.label : undefined}
                >
                  <Icon className={`w-5 h-5 ${isActive ? 'text-primary' : 'text-gray-400'}`} />
                  {!sidebarCollapsed && (
                    <span className="ml-3 whitespace-nowrap">{item.label}</span>
                  )}
                </button>
              );
            })}
          </>
        )}
      </nav>

      {/* User section */}
      <div className="border-t border-gray-200 p-3">
        <div className="flex items-center">
          <div className="w-8 h-8 bg-primary-100 rounded-full flex items-center justify-center">
            <User className="w-4 h-4 text-primary" />
          </div>
          {!sidebarCollapsed && (
            <div className="ml-3 flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-700 truncate">{user?.username || 'admin'}</p>
              <p className="text-xs text-gray-400">个人资料</p>
            </div>
          )}
          {!sidebarCollapsed && (
            <button
              onClick={handleLogout}
              className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded transition"
              title="退出登录"
            >
              <LogOut className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
