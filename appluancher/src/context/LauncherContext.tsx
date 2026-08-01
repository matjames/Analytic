import React, { createContext, useContext, useEffect, useCallback, useMemo } from 'react';
import { Application, LauncherContextType } from '@typings/index';
import { MOCK_APPLICATIONS } from '@utils/mockData';
import { useAuth } from './AuthContext';
import { filterAppsByUser } from '@utils/helpers';

const LauncherContext = createContext<LauncherContextType | undefined>(undefined);

export interface LauncherProviderProps {
  children: React.ReactNode;
  mockMode?: boolean;
}

export const LauncherProvider: React.FC<LauncherProviderProps> = ({
  children,
  mockMode = true,
}) => {
  const [applications, setApplications] = React.useState<Application[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const { user } = useAuth();

  // Fetch applications on mount
  useEffect(() => {
    const fetchApplications = async () => {
      try {
        setLoading(true);
        setError(null);

        if (mockMode) {
          // Simulate API call
          await new Promise((resolve) => setTimeout(resolve, 500));
          setApplications(MOCK_APPLICATIONS);
        } else {
          // Real API call would go here
          const response = await fetch('/api/applications');
          if (!response.ok) throw new Error('Failed to fetch applications');
          const data = await response.json();
          setApplications(data);
        }
      } catch (err) {
        const errorMsg =
          err instanceof Error ? err.message : 'Failed to load applications';
        setError(errorMsg);
        console.error('Error loading applications:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchApplications();
  }, [mockMode]);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((resolve) => setTimeout(resolve, 500));
      setApplications(MOCK_APPLICATIONS);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to refetch applications'
      );
    } finally {
      setLoading(false);
    }
  }, []);

  const getVisibleApps = useCallback(
    (userContext: typeof user) => {
      return filterAppsByUser(applications, userContext);
    },
    [applications]
  );

  const value = useMemo(
    () => ({
      applications,
      loading,
      error,
      refetch,
      getVisibleApps,
    }),
    [applications, loading, error, refetch, getVisibleApps]
  );

  return (
    <LauncherContext.Provider value={value}>
      {children}
    </LauncherContext.Provider>
  );
};

export const useLauncher = (): LauncherContextType => {
  const context = useContext(LauncherContext);
  if (!context) {
    throw new Error('useLauncher must be used within a LauncherProvider');
  }
  return context;
};
