import React, { useState, useRef, useEffect } from 'react';
import { Application } from '@typings/index';
import { X, LayoutGrid, Search } from 'lucide-react';
import { getStatusColor } from '@utils/helpers';
import { useDebouncedSearch } from '@hooks/index';

interface AppLauncherProps {
  apps: Application[];
  isOpen: boolean;
  onClose: () => void;
  onAppSelect?: (app: Application) => void;
  canAccess?: (app: Application) => boolean;
}

export const AppLauncher: React.FC<AppLauncherProps> = ({
  apps,
  isOpen,
  onClose,
  onAppSelect,
  canAccess = () => true,
}) => {
  const [filteredApps, setFilteredApps] = useState(apps);
  const launcherRef = useRef<HTMLDivElement>(null);
  const { searchTerm, setSearchTerm, debouncedTerm } = useDebouncedSearch(200);

  // Filter apps based on search
  useEffect(() => {
    if (!debouncedTerm.trim()) {
      setFilteredApps(apps);
    } else {
      const term = debouncedTerm.toLowerCase();
      setFilteredApps(
        apps.filter(
          (app) =>
            app.name.toLowerCase().includes(term) ||
            app.description.toLowerCase().includes(term) ||
            app.tags?.some((tag) => tag.toLowerCase().includes(term))
        )
      );
    }
  }, [debouncedTerm, apps]);

  // Close on escape key
  useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (
        launcherRef.current &&
        !launcherRef.current.contains(e.target as Node)
      ) {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen, onClose]);

  const handleAppClick = (app: Application) => {
    if (canAccess(app) && onAppSelect) {
      onAppSelect(app);
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black bg-opacity-50 z-40 transition-opacity" />

      {/* Launcher Modal */}
      <div
        ref={launcherRef}
        className="fixed top-16 right-4 z-50 w-full max-w-2xl bg-white rounded-lg shadow-2xl border border-gray-200 animate-slide-down"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex items-center space-x-2">
            <LayoutGrid className="w-5 h-5" style={{ color: '#165c92' }} />
            <h2 className="text-lg font-semibold text-gray-900">
              Application Launcher
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 rounded-lg transition-colors"
            aria-label="Close launcher"
          >
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* Search Bar */}
        <div className="p-4 border-b border-gray-200">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
            <input
              type="text"
              placeholder="Search applications..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              autoFocus
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Apps Grid */}
        <div className="p-4 max-h-[60vh] overflow-y-auto">
          {filteredApps.length > 0 ? (
            <div className="grid grid-cols-3 gap-3">
              {filteredApps.map((app) => (
                <button
                  key={app.id}
                  onClick={() => handleAppClick(app)}
                  disabled={!canAccess(app)}
                  className={`group flex flex-col items-center p-3 rounded-lg border transition-all duration-200 ${
                    canAccess(app)
                      ? 'border-gray-200 hover:border-blue-300 cursor-pointer'
                      : 'border-gray-100 opacity-50 cursor-not-allowed'
                  }`}
                >
                  {/* App Icon */}
                  <div className="text-2xl mb-1 group-hover:scale-110 transition-transform">
                    {app.icon}
                  </div>

                  {/* App Name */}
                  <h3 className="text-xs font-medium text-gray-900 text-center line-clamp-2">
                    {app.name}
                  </h3>

                  {/* Status */}
                  <div
                    className="mt-1 w-1.5 h-1.5 rounded-full"
                    style={{ backgroundColor: getStatusColor(app.status) }}
                    title={app.status}
                  />
                </button>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <Search className="w-12 h-12 text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500">No applications found</p>
              <p className="text-sm text-gray-400 mt-1">
                Try a different search term
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-gray-200 bg-gray-50 text-xs text-gray-500">
          <p>
            Showing {filteredApps.length} of {apps.length} applications
          </p>
        </div>
      </div>
    </>
  );
};

