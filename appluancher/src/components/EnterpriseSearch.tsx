import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Search, X, Loader2, ExternalLink } from 'lucide-react';

interface SearchResult {
  type: string;
  id: string;
  title: string;
  description: string;
  source: string;
  url: string;
  meta?: string;
  score?: number;
}

interface SearchResponse {
  query: string;
  count: number;
  results: SearchResult[];
}

const SOURCE_LABELS: Record<string, string> = {
  pms: 'Projects',
  rms: 'Research',
  registry: 'Registry',
  statchat: 'StatChat',
  helpdesk: 'HelpDesk',
};

const TYPE_ICONS: Record<string, string> = {
  project: '🏗️',
  research: '🔬',
  task: '✓',
  member: '👤',
  document: '📄',
  meeting: '📅',
  risk: '⚠️',
  issue: '🔴',
  survey: '📋',
  milestone: '🏁',
  proposal: '📝',
  ethics: '⚖️',
  grant: '💰',
  literature: '📚',
  dataset: '📊',
  publication: '📰',
  report: '📈',
  calendar: '🗓️',
  chat: '💬',
  ticket: '🎫',
  facility: '🏥',
  organization: '🏛️',
  staff: '👔',
};

interface EnterpriseSearchProps {
  onClose?: () => void;
  standalone?: boolean;
}

export const EnterpriseSearch: React.FC<EnterpriseSearchProps> = ({
  onClose,
  standalone = false,
}) => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchMode, setSearchMode] = useState<'enterprise' | 'launcher'>('enterprise');
  const searchRef = useRef<HTMLDivElement>(null);
  const debounceTimer = useRef<NodeJS.Timeout>();

  const SEARCH_API = process.env.NEXT_PUBLIC_ENTERPRISE_SEARCH_URL || 'http://localhost:8095';

  const performSearch = useCallback(async (term: string) => {
    if (term.trim().length < 2) {
      setResults([]);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`${SEARCH_API}/api/search?q=${encodeURIComponent(term)}&limit=8`);
      if (!res.ok) throw new Error('Search failed');
      const data: SearchResponse = await res.json();
      setResults(data.results || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to search enterprise');
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, [SEARCH_API]);

  useEffect(() => {
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    debounceTimer.current = setTimeout(() => {
      performSearch(query);
    }, 300);
    return () => {
      if (debounceTimer.current) clearTimeout(debounceTimer.current);
    };
  }, [query, performSearch]);

  // Close on outside click for dropdown mode
  useEffect(() => {
    if (standalone) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(e.target as Node) && onClose) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [standalone, onClose]);

  const groupedResults = results.reduce((acc, result) => {
    const source = result.source || 'other';
    if (!acc[source]) acc[source] = [];
    acc[source].push(result);
    return acc;
  }, {} as Record<string, SearchResult[]>);

  return (
    <div ref={searchRef} className={standalone ? '' : 'absolute top-full left-0 right-0 mt-2'}>
      {/* Search Input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search projects, research, staff, documents..."
          className="w-full pl-10 pr-10 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
          autoFocus={standalone}
        />
        {query && (
          <button
            onClick={() => setQuery('')}
            className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
          >
            <X className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Mode Toggle */}
      {standalone && (
        <div className="flex gap-2 mt-2">
          <button
            onClick={() => setSearchMode('enterprise')}
            className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
              searchMode === 'enterprise'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            Enterprise Wide
          </button>
          <button
            onClick={() => setSearchMode('launcher')}
            className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${
              searchMode === 'launcher'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            Applications
          </button>
        </div>
      )}

      {/* Results Dropdown */}
      {(loading || results.length > 0 || error) && (
        <div className="mt-2 bg-white rounded-lg border border-gray-200 shadow-xl max-h-96 overflow-y-auto">
          {/* Loading */}
          {loading && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
              <span className="ml-2 text-sm text-gray-500">Searching enterprise...</span>
            </div>
          )}

          {/* Error */}
          {error && !loading && (
            <div className="p-4 text-sm text-red-600">{error}</div>
          )}

          {/* Results */}
          {!loading && results.length > 0 && (
            <div className="divide-y divide-gray-100">
              {Object.entries(groupedResults).map(([source, sourceResults]) => (
                <div key={source} className="p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
                      {SOURCE_LABELS[source] || source}
                    </span>
                    <span className="text-xs text-gray-400">
                      {sourceResults.length} result{sourceResults.length !== 1 ? 's' : ''}
                    </span>
                  </div>
                  <div className="space-y-1">
                    {sourceResults.map((result, idx) => (
                      <div
                        key={`${result.source}-${result.id}-${idx}`}
                        onClick={() => {
                          if (onClose) onClose();
                          // Fallback to URL if not handled
                          if (result.url && result.url.startsWith('http')) {
                            window.open(result.url, '_blank');
                          }
                        }}
                        className="flex items-start gap-2 p-2 rounded hover:bg-blue-50 cursor-pointer transition-colors"
                      >
                        <span className="text-lg">{TYPE_ICONS[result.type] || '📄'}</span>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-1">
                            <span className="text-sm font-medium text-gray-900 truncate">
                              {result.title}
                            </span>
                            <ExternalLink className="w-3 h-3 text-gray-400 flex-shrink-0" />
                          </div>
                          {result.description && (
                            <p className="text-xs text-gray-500 truncate">{result.description}</p>
                          )}
                          {result.meta && (
                            <p className="text-xs text-gray-400">{result.meta}</p>
                          )}
                        </div>
                        <span className="text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-500 capitalize flex-shrink-0">
                          {result.type}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              ))}

              {/* Footer */}
              <div className="p-3 bg-gray-50 text-center">
                <a
                  href={`${SEARCH_API}/api/search?q=${encodeURIComponent(query)}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs text-blue-600 hover:text-blue-800"
                >
                  View all results...
                </a>
              </div>
            </div>
          )}

          {/* Empty state */}
          {!loading && !error && query.length >= 2 && results.length === 0 && (
            <div className="p-8 text-center">
              <p className="text-sm text-gray-500">No results found</p>
              <p className="text-xs text-gray-400 mt-1">Try a different search term</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default EnterpriseSearch;