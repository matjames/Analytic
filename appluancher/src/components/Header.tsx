import React from 'react';
import { Search } from 'lucide-react';

interface HeaderProps {
  onAppLauncherClick?: () => void;
}

export const Header: React.FC<HeaderProps> = ({ onAppLauncherClick }) => {

  return (
    <header className="sticky top-0 z-50 border-b-0 shadow-md" style={{ background: 'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)', boxShadow: '0 2px 8px rgba(22, 92, 146, 0.15)' }}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          {/* Logo & Brand */}
          <div className="flex items-center space-x-3">
            <img
              src="/icons/logo.png"
              alt="StatGate"
              className="w-8 h-8 rounded-lg"
              style={{ boxShadow: '0 0 0 2px rgba(255,255,255,0.2)' }}
            />
            <div className="hidden sm:block">
              <h1 className="text-lg font-bold text-white tracking-wide">
                StatGate
              </h1>
              <p className="text-xs text-white/80">Enterprise Evidence Intelligence</p>
            </div>
          </div>

          {/* Search Bar - Desktop Only */}
          <div className="hidden sm:flex flex-1 max-w-md mx-8">
            <div className="relative w-full">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Search applications..."
                className="w-full pl-10 pr-4 py-2 border border-white/50 rounded-md focus:outline-none focus:ring-2 focus:ring-white/30 focus:border-transparent text-gray-900 bg-white"
              />
            </div>
          </div>

          {/* Right Navigation */}
          <div className="flex items-center space-x-4">
            {/* App Launcher Button */}
            <button
              onClick={onAppLauncherClick}
              className="p-2 hover:bg-white/15 rounded-md transition-colors"
              aria-label="Application Launcher"
              title="Application Launcher"
            >
              <div className="w-6 h-6 grid grid-cols-3 gap-1">
                {[...Array(9)].map((_, i) => (
                  <div
                    key={i}
                    className="w-1.5 h-1.5 bg-white rounded-sm"
                  />
                ))}
              </div>
            </button>
          </div>
        </div>
      </div>
    </header>
  );
};
