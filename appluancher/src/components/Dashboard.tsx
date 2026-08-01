import React from 'react';
import { Application, AppCategory, AppStatus } from '@typings/index';
import { AppCard } from './AppCard';
import { useAuth } from '@context/AuthContext';
import { canAccessApp, groupAppsByCategory } from '@utils/helpers';
import { APP_CATEGORIES } from '@utils/mockData';
import { AlertCircle, TrendingUp } from 'lucide-react';

interface DashboardProps {
  apps: Application[];
  onAppLaunch?: (app: Application) => void;
}

export const Dashboard: React.FC<DashboardProps> = ({ apps, onAppLaunch }) => {
  const { user } = useAuth();
  const visibleApps = apps.filter((app) => canAccessApp(app, user));
  const groupedApps = groupAppsByCategory(visibleApps);

  // Count status
  const operationalCount = visibleApps.filter(
    (app) => app.status === AppStatus.OPERATIONAL
  ).length;
  const degradedCount = visibleApps.filter(
    (app) => app.status === AppStatus.DEGRADED
  ).length;

  return (
    <main className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
      {/* Hero Section */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 md:py-16">
          <div className="flex flex-col md:flex-row justify-between items-start md:items-center">
            <div>
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900">
                Welcome to StatGate Uganda
              </h2>
              <p className="text-gray-600 mt-2 max-w-2xl">
                Your central hub for statistical analysis, data management, and
                reporting. Access all your tools from one unified dashboard.
              </p>
            </div>
            <div className="mt-6 md:mt-0">
              <img
                src="https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=400&h=300&fit=crop"
                alt="Statistics"
                className="w-64 h-48 rounded-lg object-cover shadow-md"
              />
            </div>
          </div>

          {/* System Status */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-8">
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <div className="flex items-center space-x-3">
                <div className="flex-shrink-0">
                  <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
                    <TrendingUp className="w-6 h-6 text-green-600" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-gray-600">Operational Apps</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {operationalCount}
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <div className="flex items-center space-x-3">
                <div className="flex-shrink-0">
                  <div className="w-10 h-10 bg-yellow-100 rounded-lg flex items-center justify-center">
                    <AlertCircle className="w-6 h-6 text-yellow-600" />
                  </div>
                </div>
                <div>
                  <p className="text-sm text-gray-600">Degraded Apps</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {degradedCount}
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <div className="flex items-center space-x-3">
                <div className="flex-shrink-0">
                  <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
                    <span className="text-lg font-bold text-blue-600">
                      {visibleApps.length}
                    </span>
                  </div>
                </div>
                <div>
                  <p className="text-sm text-gray-600">Total Apps</p>
                  <p className="text-2xl font-bold text-gray-900">
                    Available
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Applications by Category */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        {visibleApps.length > 0 ? (
          <div className="space-y-12">
            {Object.entries(groupedApps).map(([category, categoryApps]) => {
              const categoryInfo =
                APP_CATEGORIES[category as AppCategory] ||
                APP_CATEGORIES.tools;

              return (
                <section key={category}>
                  <div className="mb-6">
                    <div
                      className="w-1 h-8 rounded-full mb-2"
                      style={{ backgroundColor: categoryInfo.color }}
                    />
                    <h3 className="text-2xl font-bold text-gray-900">
                      {categoryInfo.label}
                    </h3>
                    <p className="text-gray-600 mt-1">
                      {categoryInfo.description}
                    </p>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    {categoryApps.map((app) => (
                      <AppCard
                        key={app.id}
                        app={app}
                        onLaunch={onAppLaunch}
                        canAccess={canAccessApp(app, user)}
                      />
                    ))}
                  </div>
                </section>
              );
            })}
          </div>
        ) : (
          <div className="text-center py-12">
            <AlertCircle className="w-12 h-12 text-gray-300 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900">
              No Applications Available
            </h3>
            <p className="text-gray-600 mt-1">
              {"You don't have access to any applications yet. Contact your administrator for access."}
            </p>
          </div>
        )}
      </div>
    </main>
  );
};
