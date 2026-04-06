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
  Activity,
  ChevronRight,
  FileText,
} from 'lucide-react';
import useStore from '../store/useStore';
import { useNavigate } from 'react-router-dom';

// 菜单分组定义
const menuGroups = [
  {
    label: '运维工作台',
    items: [
      { id: 'dashboard', label: '仪表盘', icon: LayoutDashboard },
      { id: 'chat', label: '即时对话', icon: MessageSquare },
      { id: 'agents', label: '智能体', icon: Bot },
    ],
  },
  {
    label: '资源管理',
    items: [
      { id: 'cloud-platforms', label: '接入云平台', icon: Cloud },
      { id: 'resource-monitor', label: '资源监控', icon: Activity },
    ],
  },
  {
    label: '配置管理',
    items: [
      { id: 'ai-models', label: '模型配置', icon: Cpu },
      { id: 'skills', label: '技能中心', icon: Zap },
      { id: 'workflows', label: '工作流', icon: Workflow },
      { id: 'scheduled-tasks', label: '定时任务', icon: Clock },
    ],
  },
];

const adminGroup = {
  label: '系统管理',
  items: [
    { id: 'users', label: '用户管理', icon: Users },
    { id: 'operation-logs', label: '操作日志', icon: FileText },
  ],
};

export default function Sidebar() {
  const { activePage, setActivePage, user, logout, sidebarCollapsed, toggleSidebar } = useStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const handleMenuClick = (item) => {
    if (item.disabled) return;
    setActivePage(item.id);
  };

  const allGroups = user?.role === 'admin' ? [...menuGroups, adminGroup] : menuGroups;

  return (
    <div
      className={`flex flex-col transition-all duration-300 flex-shrink-0 border-r border-gray-200 ${
        sidebarCollapsed ? 'w-16' : 'w-56'
      }`}
      style={{ background: '#ffffff' }}
    >
      {/* Logo 区域 */}
      <div
        className="flex items-center h-16 px-3 flex-shrink-0 border-b border-gray-200"
      >
        <button
          onClick={toggleSidebar}
          className="flex items-center justify-center w-8 h-8 rounded-lg transition-colors flex-shrink-0 text-gray-500 hover:text-[#513CC8] hover:bg-[#EEE9FB]"
          title="折叠/展开菜单"
        >
          <Menu className="w-5 h-5" />
        </button>
        {!sidebarCollapsed && (
          <div className="ml-2 flex items-center gap-2 overflow-hidden">
            <div
              className="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0 text-sm font-bold text-white"
              style={{ background: '#513CC8' }}
            >
              AI
            </div>
            <span
              className="text-sm font-semibold whitespace-nowrap text-gray-800"
            >
              AIOPS运维平台
            </span>
          </div>
        )}
      </div>

      {/* 导航菜单 */}
      <nav className="flex-1 overflow-y-auto py-2" style={{ scrollbarWidth: 'none' }}>
        {allGroups.map((group, groupIdx) => (
          <div key={groupIdx} className="mb-1">
            {/* 分组标题 */}
            {!sidebarCollapsed && (
              <div
                className="px-4 pt-4 pb-1 text-xs uppercase tracking-widest font-medium text-gray-400"
                style={{ letterSpacing: '0.1em' }}
              >
                {group.label}
              </div>
            )}
            {sidebarCollapsed && groupIdx > 0 && (
              <div className="mx-3 my-2 border-t border-gray-100" />
            )}

            {/* 菜单项 */}
            {group.items.map((item) => {
              const Icon = item.icon;
              const isActive = activePage === item.id;
              const isDisabled = item.disabled;

              return (
                <button
                  key={item.id}
                  onClick={() => handleMenuClick(item)}
                  disabled={isDisabled}
                  title={sidebarCollapsed ? item.label : (isDisabled ? '暂无页面' : undefined)}
                  className={`w-full flex items-center h-9 text-sm transition-all duration-150 relative ${
                    sidebarCollapsed ? 'justify-center px-0' : 'px-4'
                  } ${isDisabled ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'} ${
                    isActive
                      ? 'bg-[#EEE9FB] text-[#513CC8] font-medium'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`}
                >
                  {/* 激活左边框指示 */}
                  {isActive && !sidebarCollapsed && (
                    <span
                      className="absolute left-0 top-0 bottom-0 w-0.5 rounded-r"
                      style={{ background: '#513CC8' }}
                    />
                  )}
                  <Icon className="w-4 h-4 flex-shrink-0" />
                  {!sidebarCollapsed && (
                    <span className="ml-2.5 whitespace-nowrap text-sm">{item.label}</span>
                  )}
                  {!sidebarCollapsed && isDisabled && (
                    <span
                      className="ml-auto text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-400"
                      style={{ fontSize: '10px' }}
                    >
                      即将上线
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        ))}
      </nav>

      {/* 底部用户区 */}
      <div className="flex-shrink-0 border-t border-gray-200 p-3">
        {sidebarCollapsed ? (
          <button
            onClick={handleLogout}
            className="w-full flex items-center justify-center h-9 rounded-lg transition-colors text-gray-400 hover:bg-red-50 hover:text-red-500"
            title="退出登录"
          >
            <LogOut className="w-4 h-4" />
          </button>
        ) : (
          <div className="flex items-center gap-2">
            {/* 用户头像 */}
            <div
              className="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold text-white"
              style={{ background: '#513CC8' }}
            >
              {(user?.username || 'U').slice(0, 1).toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate text-gray-700">
                {user?.username || 'admin'}
              </p>
              <span
                className="text-xs px-1.5 py-0.5 rounded"
                style={{
                  background: user?.role === 'admin' ? '#EEE9FB' : '#f3f4f6',
                  color: user?.role === 'admin' ? '#513CC8' : '#6b7280',
                  fontSize: '10px',
                }}
              >
                {user?.role === 'admin' ? '管理员' : '用户'}
              </span>
            </div>
            <button
              onClick={handleLogout}
              className="p-1.5 rounded-lg transition-colors flex-shrink-0 text-gray-400 hover:bg-red-50 hover:text-red-500"
              title="退出登录"
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
