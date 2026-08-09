import React, { useState } from 'react';
import { Application } from '@typings/index';
import { Header } from '@components/Header';
import { AppLauncher } from '@components/AppLauncher';
import { useAuth } from '@context/AuthContext';
import { useLauncher } from '@context/LauncherContext';
import { canAccessApp } from '@utils/helpers';
import { WorkspaceDashboard } from '@components/WorkspaceDashboard';

export default function Home() {
  const { user, isAuthenticated, login, logout } = useAuth();
  const { applications } = useLauncher();
  const [launcherOpen, setLauncherOpen] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loginError, setLoginError] = useState('');

  const handleAppLaunch = async (app: Application) => {
    try {
      await fetch('/api/analytics/app-launch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          appId: app.id,
          appName: app.name,
          category: app.category,
          userRole: user?.role,
          timestamp: new Date().toISOString(),
        }),
      });
    } catch (error) {
      console.error('Failed to track app launch:', error);
    }

    const token = localStorage.getItem('registry_jwt');
    let target = app.url;
    if (token && app.id !== 'app-analytics') {
      const separator = target.includes('?') ? '&' : '?';
      target = `${target}${separator}registry_token=${encodeURIComponent(token)}`;
    }
    window.location.href = target;
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoginError('');
    try {
      await login(email, password);
      setShowLogin(false);
      setEmail('');
      setPassword('');
    } catch {
      setLoginError('Invalid email or password');
    }
  };

  const renderEnterpriseWorkspace = () => {
    if (!isAuthenticated || !user) {
      return (
        <div className="min-h-screen bg-white">
          <Header onAppLauncherClick={() => setLauncherOpen(true)} />
          <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
            <div className="text-center">
              <h2 className="text-3xl font-bold text-gray-900">Welcome to StatGate</h2>
              <p className="mt-4 text-gray-600">Sign in with your Registry account to access your enterprise workspace.</p>
              <button className="mt-8 btn btn-primary" onClick={() => setShowLogin(true)}>Sign In</button>
            </div>
          </main>
          <AppLauncher apps={applications} isOpen={launcherOpen} onClose={() => setLauncherOpen(false)} onAppSelect={handleAppLaunch} canAccess={(app) => canAccessApp(app, user)} />
          {showLogin && (
            <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
              <form onSubmit={handleLogin} className="bg-white p-8 rounded-lg shadow-lg w-full max-w-md">
                <h3 className="text-xl font-semibold mb-4">Registry Sign In</h3>
                {loginError && <p className="text-red-600 mb-2">{loginError}</p>}
                <input className="mb-3 w-full border p-2 rounded" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} required />
                <input className="mb-3 w-full border p-2 rounded" type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} required />
                <div className="flex justify-end gap-2">
                  <button type="button" className="btn btn-secondary" onClick={() => setShowLogin(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary">Sign In</button>
                </div>
              </form>
            </div>
          )}
        </div>
      );
    }

    return (
      <div className="min-h-screen bg-gray-50">
        <Header onAppLauncherClick={() => setLauncherOpen(true)} />
        <AppLauncher apps={applications} isOpen={launcherOpen} onClose={() => setLauncherOpen(false)} onAppSelect={handleAppLaunch} canAccess={(app) => canAccessApp(app, user)} />
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex items-center justify-between mb-8">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Enterprise Workspace</h1>
              <p className="text-gray-600">Welcome back, {user.name}</p>
            </div>
            <button className="btn btn-secondary" onClick={logout}>Sign Out</button>
          </div>
          <WorkspaceDashboard user={user} />
        </main>
      </div>
    );
  };

  return renderEnterpriseWorkspace();
}

