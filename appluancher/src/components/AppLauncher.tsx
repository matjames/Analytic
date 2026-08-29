import React, { useState, useRef, useEffect, useMemo } from 'react';
import { Application } from '@typings/index';
import { X, LayoutGrid, Search, Compass } from 'lucide-react';
import { getStatusColor, groupAppsByCategory } from '@utils/helpers';
import { useDebouncedSearch } from '@hooks/index';
import { AppCard } from './AppCard';

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
  const [view, setView] = useState<'grid' | 'browse'>('grid');
  const [activeCategory, setActiveCategory] = useState('all');
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

  // Reset view and filters whenever the launcher (re-)opens
  useEffect(() => {
    if (isOpen) {
      setView('grid');
      setActiveCategory('all');
      setSearchTerm('');
    }
  }, [isOpen, setSearchTerm]);
  // Aggregate available categories (with counts) for the browse filter
  const categories = useMemo(() => {
    const counts: Record<string, number> = {};
    apps.forEach((app) => {
      counts[app.category] = (counts[app.category] ?? 0) + 1;
    });
    return Object.entries(counts).sort(([a], [b]) => a.localeCompare(b));
  }, [apps]);

  // Browse view applies the search + selected category together
  const browseApps = useMemo(() => {
    if (activeCategory === 'all') return filteredApps;
    return filteredApps.filter((app) => app.category === activeCategory);
  }, [filteredApps, activeCategory]);

  // Group browse results by category for the catalogue layout
  const groupedBrowseApps = useMemo(
    () => groupAppsByCategory(browseApps),
    [browseApps]
  );

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

  // Used both by the grid tiles and the browse view cards
  const handleLaunch = (app: Application) => handleAppClick(app);

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black bg-opacity-50 z-40 transition-opacity" />

      {/* Launcher Modal */}
      <div
        ref={launcherRef}
        className="fixed top-16 right-4 z-50 w-full max-w-3xl bg-white rounded-lg shadow-2xl border border-gray-200 animate-slide-down"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex items-center space-x-2">
            {view === 'grid' ? (
              <LayoutGrid className="w-5 h-5" style={{ color: '#165c92' }} />
            ) : (
              <Compass className="w-5 h-5" style={{ color: '#165c92' }} />
            )}
            <h2 className="text-lg font-semibold text-gray-900">
              {view === 'grid'
                ? 'Application Launcher'
                : 'Browse All Applications'}
            </h2>
          </div>
          <div className="flex items-center space-x-2">
            {/* View toggle */}
            <div className="flex items-center bg-gray-100 rounded-lg p-1 text-xs font-medium">
              <button
                onClick={() => setView('grid')}
                className={`px-3 py-1.5 rounded-md flex items-center space-x-1 transition-colors ${
                  view === 'grid'
                    ? 'bg-white text-blue-700 shadow-sm border border-gray-200'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                <LayoutGrid className="w-3.5 h-3.5" />
                <span>Grid</span>
              </button>
              <button
                onClick={() => setView('browse')}
                className={`px-3 py-1.5 rounded-md flex items-center space-x-1 transition-colors ${
                  view === 'browse'
                    ? 'bg-white text-blue-700 shadow-sm border border-gray-200'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                <Compass className="w-3.5 h-3.5" />
                <span>Browse All</span>
              </button>
            </div>
            <button
              onClick={onClose}
              className="p-1 hover:bg-gray-100 rounded-lg transition-colors"
              aria-label="Close launcher"
            >
              <X className="w-5 h-5 text-gray-500" />
            </button>
          </div>
        </div>

        {/* Search Bar */}
        <div className="p-4 border-b border-gray-200">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
            <input
              type="text"
              placeholder={
                view === 'grid'
                  ? 'Search applications...'
                  : 'Search the full application catalogue...'
              }
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              autoFocus
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Category Filter (Browse view only) */}
        {view === 'browse' && (
          <div className="px-4 py-3 border-b border-gray-200 bg-gray-50 flex items-center flex-wrap gap-2">
            <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide mr-1">
              Filter:
            </span>
            <button
              onClick={() => setActiveCategory('all')}
              className={`px-3 py-1 rounded-full text-xs font-medium border transition-colors ${
                activeCategory === 'all'
                  ? 'border-blue-600 bg-blue-600 text-white'
                  : 'border-gray-300 bg-white text-gray-600 hover:border-blue-300 hover:text-blue-700'
              }`}
            >
              All ({browseApps.length})
            </button>
            {categories.map(([category, count]) => (
              <button
                key={category}
                onClick={() => setActiveCategory(category)}
                className={`px-3 py-1 rounded-full text-xs font-medium border capitalize transition-colors ${
                  activeCategory === category
                    ? 'border-blue-600 bg-blue-600 text-white'
                    : 'border-gray-300 bg-white text-gray-600 hover:border-blue-300 hover:text-blue-700'
                }`}
              >
                {category} ({count})
              </button>
            ))}
          </div>
        )}

        {/* Content */}
        {view === 'grid' ? (
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

                {/* Browse All tile */}
                <button
                  onClick={() => setView('browse')}
                  className="group flex flex-col items-center justify-center p-3 rounded-lg border-2 border-dashed border-blue-300 bg-blue-50 hover:bg-blue-100 transition-all duration-200 cursor-pointer"
                >
                  <Compass className="w-6 h-6 text-blue-600 mb-1 group-hover:scale-110 transition-transform" />
                  <span className="text-xs font-semibold text-blue-700">
                    Browse All
                  </span>
                  <span className="text-[10px] text-blue-500 mt-0.5">
                    {apps.length} applications
                  </span>
                </button>
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
        ) : (
          /* Browse All view */
          <div className="p-4 max-h-[60vh] overflow-y-auto">
            {browseApps.length > 0 ? (
              activeCategory === 'all' ? (
                Object.entries(groupedBrowseApps).map(
                  ([category, appsInCategory]) => (
                    <section key={category} className="mb-6">
                      <h3 className="text-xs font-bold text-gray-700 uppercase tracking-wide mb-3 flex items-center space-x-2">
                        <span
                          className="w-2 h-2 rounded-full inline-block"
                          style={{ backgroundColor: getCategoryColor(category) }}
                        />
                        <span>{category.replace('-', ' ')}</span>
                        <span className="text-gray-400 font-medium normal-case">
                          ({appsInCategory.length})
                        </span>
                      </h3>
                      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                        {appsInCategory.map((app) => (
                          <AppCard
                            key={app.id}
                            app={app}
                            onLaunch={handleLaunch}
                            canAccess={canAccess(app)}
                          />
                        ))}
                      </div>
                    </section>
                  )
                )
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  {browseApps.map((app) => (
                    <AppCard
                      key={app.id}
                      app={app}
                      onLaunch={handleLaunch}
                      canAccess={canAccess(app)}
                    />
                  ))}
                </div>
              )
            ) : (
              <div className="text-center py-12">
                <Search className="w-12 h-12 text-gray-300 mx-auto mb-4" />
                <p className="text-gray-500">No applications found</p>
                <p className="text-sm text-gray-400 mt-1">
                  Try a different search term or category
                </p>
              </div>
            )}
          </div>
        )}

        {/* Footer */}
        <div className="p-4 border-t border-gray-200 bg-gray-50 text-xs text-gray-500 flex items-center justify-between">
          <p>
            {view === 'grid'
              ? `Showing ${filteredApps.length} of ${apps.length} applications`
              : `Browsing ${browseApps.length} of ${apps.length} applications`}
          </p>
          <div className="flex items-center gap-2">
            {view === 'grid' ? (
              <button
                type="button"
                onClick={() => setView('browse')}
                className="px-3 py-1.5 rounded-md text-white text-xs font-medium transition-opacity hover:opacity-90 cursor-pointer flex items-center space-x-1.5"
                style={{
                  background:
                    'linear-gradient(135deg, #165c92 0%, #1a7ab5 100%)',
                }}
              >
                <Compass className="w-3.5 h-3.5" />
                <span>Browse All Applications</span>
              </button>
            ) : (
              <button
                type="button"
                onClick={() => setView('grid')}
                className="px-3 py-1.5 rounded-md border border-gray-300 bg-white text-gray-600 text-xs font-medium hover:border-blue-300 hover:text-blue-700 transition-colors flex items-center space-x-1.5"
              >
                <LayoutGrid className="w-3.5 h-3.5" />
                <span>Back to Grid</span>
              </button>
            )}
          </div>
        </div>
      </div>
    </>
  );
};

// Small helper to colour the category headings in the browse view
function getCategoryColor(category: string): string {
  const colors: Record<string, string> = {
    analytics: '#0ea5e9',
    database: '#06b6d4',
    reporting: '#8b5cf6',
    admin: '#ef4444',
    tools: '#f59e0b',
    integration: '#10b981',
    management: '#6366f1',
  };
  return colors[category] ?? '#64748b';
}

