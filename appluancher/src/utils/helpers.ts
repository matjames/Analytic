import { User, UserRole, Application, AppStatus } from '@typings/index';

/**
 * Filter applications based on user permissions
 */
export const filterAppsByUser = (
  applications: Application[],
  user: User | null
): Application[] => {
  if (!user) {
    return applications.filter((app) =>
      app.requiredRoles.includes(UserRole.GUEST)
    );
  }

  return applications.filter((app) =>
    app.requiredRoles.includes(user.role)
  );
};

/**
 * Get status color based on app status
 */
export const getStatusColor = (status: AppStatus): string => {
  const colors: Record<AppStatus, string> = {
    [AppStatus.OPERATIONAL]: '#10b981', // green
    [AppStatus.DEGRADED]: '#f59e0b', // yellow
    [AppStatus.MAINTENANCE]: '#3b82f6', // blue
    [AppStatus.OFFLINE]: '#ef4444', // red
  };
  return colors[status];
};

/**
 * Format date to readable string
 */
export const formatDate = (date: Date | string | undefined): string => {
  if (!date) return 'Unknown';
  const d = new Date(date);
  return d.toLocaleDateString('en-UG', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
};

/**
 * Get time since last update
 */
export const getTimeSinceUpdate = (date: Date | string | undefined): string => {
  if (!date) return 'Unknown';
  const d = new Date(date);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - d.getTime()) / 1000);

  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
  return formatDate(date);
};

/**
 * Check if app is accessible by user
 */
export const canAccessApp = (app: Application, user: User | null): boolean => {
  if (!user) {
    return app.requiredRoles.includes(UserRole.GUEST);
  }
  return app.requiredRoles.includes(user.role);
};

/**
 * Group applications by category
 */
export const groupAppsByCategory = (
  apps: Application[]
): Record<string, Application[]> => {
  return apps.reduce(
    (acc, app) => {
      if (!acc[app.category]) {
        acc[app.category] = [];
      }
      acc[app.category].push(app);
      return acc;
    },
    {} as Record<string, Application[]>
  );
};