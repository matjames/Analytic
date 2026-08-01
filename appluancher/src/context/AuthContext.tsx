import React, { createContext, useContext, useCallback, useMemo } from 'react';
import { User, UserRole, AuthContext as AuthContextType } from '@typings/index';
import { MOCK_USER } from '@utils/mockData';

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export interface AuthProviderProps {
  children: React.ReactNode;
  mockMode?: boolean;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({
  children,
  mockMode = true,
}) => {
  const [user, setUser] = React.useState<User | null>(
    mockMode ? MOCK_USER : null
  );
  const [isLoading, setIsLoading] = React.useState(false);

  const login = useCallback(async (email: string, _password: string) => {
    setIsLoading(true);
    try {
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 500));
      setUser({
        ...MOCK_USER,
        email,
        lastLogin: new Date(),
      });
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    setIsLoading(true);
    try {
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 300));
      setUser(null);
    } finally {
      setIsLoading(false);
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
