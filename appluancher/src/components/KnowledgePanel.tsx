import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

interface KnowledgeResult {
  id: string;
  type: string;
  title: string;
  summary?: string;
  source: string;
  deep_link: string;
}

// A compact, permission-aware entry point to the Enterprise Knowledge home.
// It deliberately follows deep links to source applications rather than
// copying their records into the launcher.
export const KnowledgePanel: React.FC<{ user: User }> = ({ user }) => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<KnowledgeResult[]>([]);
  const [error, setError] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    const headers = { 'X-User-ID': user.id };
    fetch(`${API_BASE}/api/knowledge/search?q=${encodeURIComponent(query)}`, { headers, signal: controller.signal })
      .then(response => response.ok ? response.json() : Promise.reject())
      .then(payload => { setResults(payload.results || []); setError(false); })
      .catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => controller.abort();
  }, [query, user.id]);

  return (
    <section className="glass-panel">
      <div className="flex items-center justify-between gap-2 mb-3">
        <h3 className="text-lg font-semibold">Knowledge</h3>
        <span className="text-xs text-gray-500">Enterprise memory</span>
      </div>
      <input
        aria-label="Search enterprise knowledge"
        className="w-full border rounded px-3 py-2 text-sm mb-3"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Search lessons, decisions, definitions"
      />
      {error ? <p className="text-sm text-gray-500">Knowledge is currently unavailable.</p> : (
        <ul className="space-y-2">
          {results.length === 0 && <li className="text-sm text-gray-500">No permitted knowledge found.</li>}
          {results.slice(0, 5).map(result => (
            <li key={`${result.type}-${result.id}`} className="border-b pb-2 last:border-0">
              <a className="block hover:text-blue-700" href={result.deep_link}>
                <div className="text-sm font-medium truncate">{result.title}</div>
                <div className="text-xs text-gray-500 truncate">{result.type.replace('_', ' ')} · {result.source}</div>
              </a>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};
