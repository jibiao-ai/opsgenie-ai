import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { login } from '../services/api';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';
import { Bot, Cloud, Shield, Cpu, Zap, Lock, User, Eye, EyeOff } from 'lucide-react';

export default function LoginPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const setAuth = useStore((s) => s.setAuth);
  const navigate = useNavigate();

  const handleLogin = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const res = await login(username, password);
      if (res.code === 0) {
        setAuth(res.data.user, res.data.token);
        toast.success('登录成功');
        navigate('/', { replace: true });
      } else {
        toast.error(res.message || '登录失败');
      }
    } catch (err) {
      toast.error('登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
    }
  };

  const features = [
    { icon: Cloud,  label: '多云统一管理', desc: '支持 EasyStack、ZStack 等主流私有云平台' },
    { icon: Cpu,    label: 'AI 驱动运维',   desc: '集成大模型 API，智能分析运维问题' },
    { icon: Zap,    label: '自动化工作流',  desc: '灵活编排运维任务，提升响应效率' },
  ];

  return (
    <div className="min-h-screen flex">
      {/* 左侧装饰区（40%）*/}
      <div
        className="hidden md:flex md:w-2/5 flex-col justify-between p-10 relative overflow-hidden"
        style={{ background: '#513CC8' }}
      >
        {/* 背景装饰圆 */}
        <div
          className="absolute -top-24 -right-24 w-72 h-72 rounded-full opacity-15"
          style={{ background: '#ffffff' }}
        />
        <div
          className="absolute -bottom-16 -left-16 w-56 h-56 rounded-full opacity-15"
          style={{ background: '#ffffff' }}
        />

        {/* Logo 区域 */}
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-8">
            <img src="/logo-icon.png" alt="OpsGenie AI" className="w-12 h-12 rounded-2xl" style={{ background: 'rgba(255,255,255,0.15)' }} />
            <div>
              <h1 className="text-xl font-bold text-white">OpsGenie <span className="text-sm font-normal px-1.5 py-0.5 rounded" style={{ background: 'rgba(255,255,255,0.2)' }}>AI</span></h1>
              <p className="text-xs" style={{ color: 'rgba(255,255,255,0.65)' }}>Intelligent Cloud Operations Platform</p>
            </div>
          </div>

          <div className="mb-2">
            <h2 className="text-3xl font-bold text-white mb-3 leading-snug">
              智能化云基础设施<br />运维管理平台
            </h2>
            <p className="text-sm leading-relaxed" style={{ color: 'rgba(255,255,255,0.85)' }}>
              集 AI 智能体、多云接入、自动化工作流于一体，让运维更简单、更高效。
            </p>
          </div>
        </div>

        {/* 功能亮点 */}
        <div className="relative z-10 space-y-4">
          {features.map((f, i) => {
            const Icon = f.icon;
            return (
              <div key={i} className="flex items-start gap-4">
                <div
                  className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0"
                  style={{ background: 'rgba(255,255,255,0.15)' }}
                >
                  <Icon className="w-4 h-4" style={{ color: '#ffffff' }} />
                </div>
                <div>
                  <p className="text-sm font-medium text-white">{f.label}</p>
                  <p className="text-xs mt-0.5" style={{ color: 'rgba(255,255,255,0.65)' }}>{f.desc}</p>
                </div>
              </div>
            );
          })}
        </div>

        {/* 版权 */}
        <p className="relative z-10 text-xs" style={{ color: 'rgba(255,255,255,0.5)' }}>
          © 2024 OpsGenie AI. All rights reserved.
        </p>
      </div>

      {/* 右侧登录区（60%）*/}
      <div className="flex-1 flex flex-col items-center justify-center bg-white px-8 py-12">
        {/* 移动端 Logo */}
        <div className="md:hidden mb-8 text-center">
          <img src="/logo-icon.png" alt="OpsGenie AI" className="w-14 h-14 rounded-2xl mx-auto mb-3" />
          <h1 className="text-xl font-bold text-gray-800"><span>Ops</span><span style={{ color: '#513CC8' }}>Genie</span> <span className="text-sm px-1.5 py-0.5 rounded" style={{ background: '#EEE9FB', color: '#513CC8' }}>AI</span></h1>
        </div>

        {/* 登录表单容器 */}
        <div className="w-full max-w-sm">
          <div className="mb-8">
            <h2 className="text-2xl font-bold text-gray-800 mb-1">欢迎回来</h2>
            <p className="text-sm text-gray-400">请输入账号和密码登录平台</p>
          </div>

          <form onSubmit={handleLogin} className="space-y-5">
            {/* 用户名 */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">
                用户名
              </label>
              <div className="relative">
                <User className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none transition"
                  style={{ '--tw-ring-color': '#513CC8' }}
                  onFocus={e => { e.target.style.borderColor = '#513CC8'; e.target.style.boxShadow = '0 0 0 3px rgba(81,60,200,0.1)'; }}
                  onBlur={e => { e.target.style.borderColor = '#e5e7eb'; e.target.style.boxShadow = 'none'; }}
                  placeholder="请输入用户名"
                  required
                />
              </div>
            </div>

            {/* 密码 */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">
                密码
              </label>
              <div className="relative">
                <Lock className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full pl-10 pr-10 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none transition"
                  onFocus={e => { e.target.style.borderColor = '#513CC8'; e.target.style.boxShadow = '0 0 0 3px rgba(81,60,200,0.1)'; }}
                  onBlur={e => { e.target.style.borderColor = '#e5e7eb'; e.target.style.boxShadow = 'none'; }}
                  placeholder="请输入密码"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  tabIndex={-1}
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              <p className="text-xs text-gray-400 mt-1">密码须 ≥9位，含大小写字母、数字和特殊字符</p>
            </div>

            {/* 登录按钮 */}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 rounded-xl text-white font-medium text-sm transition-all disabled:opacity-50 disabled:cursor-not-allowed mt-2"
              style={{ background: loading ? '#7c6dd4' : '#513CC8' }}
              onMouseEnter={e => { if (!loading) e.currentTarget.style.background = '#4231a8'; }}
              onMouseLeave={e => { if (!loading) e.currentTarget.style.background = '#513CC8'; }}
            >
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  登录中...
                </span>
              ) : (
                '登 录'
              )}
            </button>
          </form>

          {/* 底部信息 */}
          <div className="mt-8 pt-6 border-t border-gray-100">
            <div className="flex items-center justify-center gap-6 text-xs text-gray-400">
              <div className="flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5" />
                <span>安全可信</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Cloud className="w-3.5 h-3.5" />
                <span>云原生架构</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Bot className="w-3.5 h-3.5" />
                <span>AI 驱动</span>
              </div>
            </div>
            <p className="text-center text-xs text-gray-300 mt-4">
              Powered by OpsGenie AI v3.1.0
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
