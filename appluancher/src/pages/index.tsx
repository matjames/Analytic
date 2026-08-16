import React, { useState } from 'react';
import { Application } from '@typings/index';
import { Header } from '@components/Header';
import { AppLauncher } from '@components/AppLauncher';
import { useAuth } from '@context/AuthContext';
import { useLauncher } from '@context/LauncherContext';
import { canAccessApp } from '@utils/helpers';
import { CommandCentre } from '@components/CommandCentre';

export default function Home() {
  const { user, isAuthenticated, login } = useAuth();
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
      // Use one canonical hand-off parameter. Every upgraded module accepts the
      // legacy registry_token name when visited directly, but duplicating a JWT
      // in the URL increases its exposure to browser history and access logs.
      target = `${target}${separator}statgate_token=${encodeURIComponent(token)}`;
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
          <Header onAppLauncherClick={() => setLauncherOpen(true)} user={null} />
          <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
            <div className="text-center">
              <h2 className="text-3xl font-bold text-gray-900">Welcome to StatGate</h2>
              <p className="mt-4 text-gray-600">
                Sign in with your Registry account to access your enterprise workspace and command centre.
              </p>
              <button className="mt-8 btn btn-primary" onClick={() => setShowLogin(true)}>
                Sign In
              </button>
            </div>
          </main>
          <AppLauncher
            apps={applications}
            isOpen={launcherOpen}
            onClose={() => setLauncherOpen(false)}
            onAppSelect={handleAppLaunch}
            canAccess={(app) => canAccessApp(app, user)}
          />
          {showLogin && (
            <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
              <form
                onSubmit={handleLogin}
                className="bg-white p-8 rounded-xl shadow-2xl w-full max-w-md border border-gray-100"
              >
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-xl font-bold text-gray-900">Sign In to StatGate</h3>
                  <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded font-semibold">SSO Gateway</span>
                </div>
                {loginError && <p className="text-red-600 text-xs mb-3 bg-red-50 p-2 rounded">{loginError}</p>}
                
                <div className="mb-3">
                  <label className="block text-xs font-semibold text-gray-700 mb-1">Email / Username</label>
                  <input
                    className="w-full border border-gray-300 p-2.5 rounded-lg text-sm outline-none focus:border-blue-600"
                    placeholder="e.g. admin@statgate.gov"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                  />
                </div>

                <div className="mb-4">
                  <label className="block text-xs font-semibold text-gray-700 mb-1">Password</label>
                  <input
                    className="w-full border border-gray-300 p-2.5 rounded-lg text-sm outline-none focus:border-blue-600"
                    type="password"
                    placeholder="Enter your password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>

                <div className="flex justify-end gap-2 mb-6">
                  <button
                    type="button"
                    className="px-4 py-2 text-xs font-semibold text-gray-600 hover:bg-gray-100 rounded-lg transition"
                    onClick={() => setShowLogin(false)}
                  >
                    Cancel
                  </button>
                  <button type="submit" className="px-5 py-2 text-xs font-semibold bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition shadow-sm">
                    Sign In
                  </button>
                </div>

                {/* Quick 1-Click Demo Personas */}
                <div className="pt-4 border-t border-gray-200">
                  <p className="text-xs font-bold text-gray-500 mb-2 uppercase tracking-wider">⚡ 1-Click Quick Demo Sign-In</p>
                  <div className="grid grid-cols-3 gap-2">
                    <button
                      type="button"
                      onClick={() => login('admin@statgate.gov', 'admin123').then(() => setShowLogin(false))}
                      className="p-2 text-center bg-blue-50 hover:bg-blue-100 text-blue-900 rounded-lg border border-blue-200 text-xs font-semibold transition"
                    >
                      🛡️ Admin
                    </button>
                    <button
                      type="button"
                      onClick={() => login('analyst@statgate.gov', 'analyst123').then(() => setShowLogin(false))}
                      className="p-2 text-center bg-emerald-50 hover:bg-emerald-100 text-emerald-900 rounded-lg border border-emerald-200 text-xs font-semibold transition"
                    >
                      📈 Analyst
                    </button>
                    <button
                      type="button"
                      onClick={() => login('manager@statgate.gov', 'manager123').then(() => setShowLogin(false))}
                      className="p-2 text-center bg-purple-50 hover:bg-purple-100 text-purple-900 rounded-lg border border-purple-200 text-xs font-semibold transition"
                    >
                      📋 Manager
                    </button>
                  </div>
                </div>
              </form>

            </div>
          )}
        </div>
      );
    }

    return (
      <div className="min-h-screen bg-gray-50 flex flex-col">
        <Header onAppLauncherClick={() => setLauncherOpen(true)} user={user} />
        <AppLauncher
          apps={applications}
          isOpen={launcherOpen}
          onClose={() => setLauncherOpen(false)}
          onAppSelect={handleAppLaunch}
          canAccess={(app) => canAccessApp(app, user)}
        />
        <CommandCentre user={user} />
      </div>
    );
  };

  return renderEnterpriseWorkspace();
}


