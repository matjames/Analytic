import React, { useState, useEffect, useRef } from 'react';
import { Search, Bell, LogOut, Shield, ChevronDown, Sparkles } from 'lucide-react';
import { EnterpriseSearch } from './EnterpriseSearch';
import { AICopilotModal } from './AICopilotModal';
import { User } from '@typings/index';
import { useAuth } from '@context/AuthContext';

interface HeaderProps {
  onAppLauncherClick?: () => void;
  user?: User | null;
  branding?: {
    display_name?: string;
    logo_url?: string;
    primary_color?: string;
    secondary_color?: string;
  } | null;
}

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const Header: React.FC<HeaderProps> = ({ onAppLauncherClick, user, branding }) => {
  const { logout } = useAuth();
  const [searchOpen, setSearchOpen] = useState(false);
  const [aiModalOpen, setAiModalOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState<number>(0);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [confirmSignOut, setConfirmSignOut] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!user) return;
    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = { 'X-User-ID': user.id };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/notifications?user_id=${encodeURIComponent(user.id)}`, { headers })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((data) => {
        if (data && Array.isArray(data.notifications)) {
          const unread = data.notifications.filter((n: any) => !n.read).length;
          setUnreadCount(unread);
        }
      });
  }, [user]);

  // Click outside to close user dropdown
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSignOut = () => {
    setConfirmSignOut(false);
    setUserMenuOpen(false);
    logout();
  };

  return (
    <>
      <header
        className="sticky top-0 z-40 border-b-0 shadow-md"
        style={{
          background: `linear-gradient(135deg, ${branding?.primary_color || '#0f766e'} 0%, ${branding?.secondary_color || '#0f3d3e'} 100%)`,
          boxShadow: '0 2px 8px rgba(15, 61, 62, 0.2)',
        }}
      >
        <div className="w-full px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo & Brand */}
            <div className="flex items-center space-x-3">
              <img
                src={branding?.logo_url || '/icons/logo.png'}
                alt={branding?.display_name || 'StatGate'}
                width={32}
                height={32}
                className="w-8 h-8 rounded-lg object-cover"
                style={{ boxShadow: '0 0 0 2px rgba(255,255,255,0.2)' }}
              />
              <div className="hidden sm:block">
                <h1 className="text-lg font-bold text-white tracking-wide">{branding?.display_name || 'StatGate'}</h1>
                <p className="text-xs text-white/80">Enterprise Evidence Intelligence</p>
              </div>
            </div>

            {/* Enterprise Search - Desktop Only */}
            <div className="hidden sm:flex flex-1 max-w-xl mx-8 relative">
              <button onClick={() => setSearchOpen(!searchOpen)} className="w-full text-left">
                <div className="relative w-full">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
                  <div className="w-full pl-10 pr-4 py-2 border border-white/50 rounded-md text-left text-sm text-gray-500 bg-white shadow-inner">
                    Search projects, facilities, surveys, decisions...
                  </div>
                </div>
              </button>
              {searchOpen && (
                <div className="relative z-50 pt-2">
                  <EnterpriseSearch onClose={() => setSearchOpen(false)} />
                </div>
              )}
            </div>

            {/* Right Navigation */}
            <div className="flex items-center space-x-3">
              {/* Governed AI Copilot Button */}
              <button
                onClick={() => setAiModalOpen(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-blue-500/20 hover:bg-blue-500/30 border border-blue-400/30 text-white text-xs font-semibold transition-all shadow-sm"
                title="Open Governed AI Copilot"
              >
                <Sparkles className="w-3.5 h-3.5 text-blue-300 animate-pulse" />
                <span className="hidden sm:inline">AI Copilot</span>
              </button>

              {/* Notifications Bell */}
              {user && (
                <button
                  className="p-2 hover:bg-white/15 rounded-md transition-colors relative text-white"
                  aria-label="Notifications"
                  title={`${unreadCount} unread notifications`}
                >
                  <Bell className="w-5 h-5 text-white" />
                  {unreadCount > 0 && (
                    <span className="absolute top-1 right-1 px-1.5 py-0.2 min-w-[16px] h-4 text-[10px] font-bold rounded-full bg-red-500 text-white flex items-center justify-center">
                      {unreadCount > 9 ? '9+' : unreadCount}
                    </span>
                  )}
                </button>
              )}

              {/* App Launcher Button */}
              <button
                onClick={onAppLauncherClick}
                className="p-2 hover:bg-white/15 rounded-md transition-colors"
                aria-label="Application Launcher"
                title="Application Launcher"
              >
                <div className="w-6 h-6 grid grid-cols-3 gap-1">
                  {[...Array(9)].map((_, i) => (
                    <div key={i} className="w-1.5 h-1.5 bg-white rounded-sm" />
                  ))}
                </div>
              </button>

              {/* User Identity & Dropdown */}
              {user && (
                <div className="relative" ref={menuRef}>
                  <button
                    onClick={() => setUserMenuOpen(!userMenuOpen)}
                    className="flex items-center gap-2 pl-2 pr-3 py-1.5 rounded-full hover:bg-white/15 transition-colors text-white text-sm"
                  >
                    <div className="w-7 h-7 rounded-full bg-white text-blue-900 font-bold flex items-center justify-center text-xs shadow">
                      {user.name ? user.name.slice(0, 2).toUpperCase() : 'SG'}
                    </div>
                    <span className="hidden md:inline font-medium text-xs truncate max-w-[120px]">
                      {user.name}
                    </span>
                    <ChevronDown className="w-3.5 h-3.5 text-white/80" />
                  </button>

                  {userMenuOpen && (
                    <div className="absolute right-0 mt-2 w-64 bg-white rounded-xl shadow-2xl border border-gray-100 py-2 z-50 animate-in fade-in slide-in-from-top-2 duration-150">
                      <div className="px-4 py-3 border-b border-gray-100">
                        <div className="font-semibold text-gray-900 text-sm">{user.name}</div>
                        <div className="text-xs text-gray-500 truncate">{user.email}</div>
                        <div className="mt-2 flex items-center gap-1.5">
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-blue-50 text-blue-700 uppercase tracking-wider flex items-center gap-1">
                            <Shield className="w-3 h-3" />
                            {user.role}
                          </span>
                        </div>
                      </div>

                      <div className="py-1">
                        <button
                          onClick={() => setConfirmSignOut(true)}
                          className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 flex items-center gap-2 transition-colors font-medium"
                        >
                          <LogOut className="w-4 h-4 text-red-500" />
                          Sign Out of StatGate
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Sign Out Confirmation Modal */}
      {confirmSignOut && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full p-6 space-y-4 animate-in fade-in zoom-in-95">
            <div className="flex items-center gap-3 text-red-600">
              <div className="p-3 bg-red-50 rounded-xl">
                <LogOut className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-lg font-bold text-gray-900">Sign Out Confirmation</h3>
                <p className="text-xs text-gray-500">Institutional Session Termination</p>
              </div>
            </div>
            <p className="text-sm text-gray-600">
              Are you sure you want to end your active enterprise session? All live real-time streams and cached session data will be safely terminated.
            </p>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setConfirmSignOut(false)}
                className="btn btn-secondary text-sm"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSignOut}
                className="btn btn-danger text-sm"
              >
                Confirm Sign Out
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Governed AI Copilot Modal */}
      <AICopilotModal isOpen={aiModalOpen} onClose={() => setAiModalOpen(false)} />
    </>
  );
};


