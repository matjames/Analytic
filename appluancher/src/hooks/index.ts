import { useEffect, useState } from 'react';

/**
 * Hook for debounced search
 */
export const useDebouncedSearch = (delay = 300) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [debouncedTerm, setDebouncedTerm] = useState('');

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedTerm(searchTerm);
    }, delay);

    return () => clearTimeout(timer);
  }, [searchTerm, delay]);

  return { searchTerm, setSearchTerm, debouncedTerm };
};