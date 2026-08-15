import React, { createContext, useContext, useCallback, useMemo, useEffect } from 'react';
import { User, UserRole, AuthContext as AuthContextType } from '@typings/index';

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const REGISTRY_LOGIN_URL = 'http://localhost:9090/api/users/login';
const REGISTRY_ME_URL = 'http://localhost:9090/api/users/me';

export interface AuthProviderProps {
  children: React.ReactNode;
  mockMode?: boolean;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({
  children,
  mockMode = false,
}) => {
  const [user, setUser] = React.useState<User | null>(null);
  const [isLoading, setIsLoading] = React.useState(false);

  useEffect(() => {
    if (mockMode) {
      return;
    }
    const token = localStorage.getItem('registry_jwt');
    if (!token) {
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch(REGISTRY_ME_URL, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!res.ok) {
          localStorage.removeItem('registry_jwt');
          return;
        }
        const data = await res.json();
        const u = data.user ?? data;
        if (!cancelled) {
          setUser({
            id: u.id?.toString() ?? u.userId?.toString() ?? token,
            name: u.name ?? `${u.first_name ?? ''} ${u.last_name ?? ''}`.trim() ?? u.username ?? 'Registry User',
            email: u.email ?? '',
            role: (u.role as UserRole) ?? UserRole.ANALYST,
            permissions: u.permissions ?? [],
            department: u.department ?? '',
            lastLogin: new Date(),
          });
        }
      } catch {
        localStorage.removeItem('registry_jwt');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [mockMode]);

  const login = useCallback(async (email: string, password: string) => {
    setIsLoading(true);
    try {
      const res = await fetch(REGISTRY_LOGIN_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ emailOrUsername: email, password }),
      });
      if (!res.ok) {
        throw new Error('Registry login failed');
      }
      const data = await res.json();
      const token = data.token ?? data.access_token;
      if (!token) {
        throw new Error('Missing token');
      }
      localStorage.setItem('registry_jwt', token);
      const u = data.user ?? data;
      setUser({
        id: u.id?.toString() ?? u.userId?.toString() ?? 'user-001',
        name: u.name ?? `${u.first_name ?? ''} ${u.last_name ?? ''}`.trim() ?? u.username ?? 'Registry User',
        email: u.email ?? email,
        role: (u.role as UserRole) ?? UserRole.ANALYST,
        permissions: u.permissions ?? [],
        department: u.department ?? '',
        lastLogin: new Date(),
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    localStorage.removeItem('registry_jwt');
    localStorage.removeItem('statgate_active_tab');
    localStorage.removeItem('statgate_user_prefs');
    sessionStorage.clear();
    setUser(null);
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new Event('statgate-logout'));
    }
  }, []);

  const hasPermission = useCallback(
    (permission: string): boolean => {
      if (!user) return false;
      return user.permissions.includes(permission);
    },
    [user]
  );

  const hasRole = useCallback(
    (role: UserRole | UserRole[]): boolean => {
      if (!user) return false;
      const rolesArray = Array.isArray(role) ? role : [role];
      return rolesArray.includes(user.role);
    },
    [user]
  );

  const value = useMemo(
    () => ({
      user,
      isAuthenticated: !!user,
      isLoading,
      login,
      logout,
      hasPermission,
      hasRole,
    }),
    [user, isLoading, login, logout, hasPermission, hasRole]
  );

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
