// Type definitions for the application

export enum UserRole {
  ADMIN = 'admin',
  MANAGER = 'manager',
  ANALYST = 'analyst',
  VIEWER = 'viewer',
  GUEST = 'guest',
}

export interface User {
  id: string;
  name: string;
  email: string;
  avatar?: string;
  role: UserRole;
  permissions: string[];
  department?: string;
  lastLogin?: Date;
}

export interface Application {
  id: string;
  name: string;
  description: string;
  icon: string;
  url: string;
  category: AppCategory;
  status: AppStatus;
  requiredRoles: UserRole[];
  version?: string;
  lastUpdated?: Date;
  healthEndpoint?: string;
  documentation?: string;
  tags?: string[];
}

export enum AppCategory {
  ANALYTICS = 'analytics',
  DATABASE = 'database',
  REPORTING = 'reporting',
  ADMIN = 'admin',
  TOOLS = 'tools',
  INTEGRATION = 'integration',
}

export enum AppStatus {
  OPERATIONAL = 'operational',
  MAINTENANCE = 'maintenance',
  DEGRADED = 'degraded',
  OFFLINE = 'offline',
}

export interface AuthContext {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  hasPermission: (permission: string) => boolean;
  hasRole: (role: UserRole | UserRole[]) => boolean;
}

export interface LauncherContextType {
  applications: Application[];
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  getVisibleApps: (user: User | null) => Application[];
}
