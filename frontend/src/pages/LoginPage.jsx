import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { login } from '../services/api';
import useStore from '../store/useStore';
import toast from 'react-hot-toast';
import { Bot, Cloud, Shield } from 'lucide-react';

export default function LoginPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
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

  return (
    <div className="min-h-screen bg-gradient-to-br from-primary-600 via-primary-700 to-primary-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo Section */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-white/20 backdrop-blur rounded-2xl mb-4">
            <Bot className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-3xl font-bold text-white mb-2">AIOPS智能运维平台</h1>
          <p className="text-primary-200">智能化云基础设施运维管理</p>
        </div>

        {/* Login Card */}
        <div className="bg-white rounded-2xl shadow-2xl p-8">
          <h2 className="text-xl font-semibold text-gray-800 mb-6 text-center">账号登录</h2>
          <form onSubmit={handleLogin}>
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-600 mb-2">用户名</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition"
                placeholder="请输入用户名"
                required
              />
            </div>
            <div className="mb-6">
              <label className="block text-sm font-medium text-gray-600 mb-2">密码</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 border border-gray-200 rounded-xl focus:ring-2 focus:ring-primary focus:border-transparent outline-none transition"
                placeholder="请输入密码（≥9位，含大小写字母、数字和特殊字符）"
                required
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-primary hover:bg-primary-600 text-white py-3 rounded-xl font-medium transition disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? '登录中...' : '登 录'}
            </button>
          </form>

          <div className="mt-6 flex items-center justify-center gap-6 text-xs text-gray-400">
            <div className="flex items-center gap-1">
              <Cloud className="w-3 h-3" />
              <span>云原生架构</span>
            </div>
            <div className="flex items-center gap-1">
              <Shield className="w-3 h-3" />
              <span>安全可信</span>
            </div>
            <div className="flex items-center gap-1">
              <Bot className="w-3 h-3" />
              <span>AI 驱动</span>
            </div>
          </div>
        </div>

        <p className="text-center text-primary-200/60 text-xs mt-6">
          Powered by AIOPS Platform
        </p>
      </div>
    </div>
  );
}
