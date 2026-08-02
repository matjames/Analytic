import { useEffect, useMemo, useRef, useState } from 'react';
import type { SidebarView } from '../App';

interface Item {
  id: SidebarView;
  label: string;
  icon: string;
}

interface Props {
  activeView: SidebarView;
  onSelect: (view: SidebarView) => void;
  theme: 'light' | 'dark';
}

const navItems: Item[] = [
  { id: 'directMessages', label: 'Direct Messages', icon: '💬' },
  { id: 'teams', label: 'Teams', icon: '👥' },
  { id: 'channels', label: 'Channels', icon: '#️⃣' },
  { id: 'meetings', label: 'Meetings', icon: '📅' },
  { id: 'wellness', label: 'Wellness', icon: '🧠' },
  { id: 'knowledge', label: 'Knowledge', icon: '📚' },
];

const ICON_SIZE = 32;
const ICON_FONT_SIZE = 18;
const GAP = 6;
const SIDEBAR_WIDTH = 68;
const MIN_VISIBLE_ITEMS = 5;

export default function SidebarRail({ activeView, onSelect, theme }: Props) {
  const railBackground = theme === 'dark' ? '#0a2b45' : '#f9fafb';
  const accent = theme === 'dark' ? '#165c92' : '#165c92';
  const textColor = theme === 'dark' ? '#e8eef4' : '#1a1a1a';
  const [maxVisible, setMaxVisible] = useState(navItems.length);
  const [overflowOpen, setOverflowOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const visibleCount = useMemo(() => {
    if (navItems.length <= maxVisible) {
      return navItems.length;
    }
    return Math.max(MIN_VISIBLE_ITEMS, maxVisible - 1); // reserve slot for More
  }, [maxVisible]);

  const hasOverflow = navItems.length > visibleCount;
  const visibleItems = hasOverflow ? navItems.slice(0, visibleCount) : navItems;
  const overflowItems = hasOverflow ? navItems.slice(visibleCount) : [];

  useEffect(() => {
    const updateMaxVisible = () => {
      if (!containerRef.current) {
        return;
      }
      const availableHeight = containerRef.current.clientHeight;
      const itemArea = ICON_SIZE + GAP;
      const computed = Math.floor((availableHeight - ICON_SIZE * 1.5) / itemArea);
      setMaxVisible(Math.max(MIN_VISIBLE_ITEMS, Math.min(navItems.length, computed)));
    };

    updateMaxVisible();
    window.addEventListener('resize', updateMaxVisible);
    return () => window.removeEventListener('resize', updateMaxVisible);
  }, []);

  useEffect(() => {
    if (!hasOverflow) {
      setOverflowOpen(false);
    }
  }, [hasOverflow]);

  return (
    <aside
      style={{
        width: SIDEBAR_WIDTH,
        minWidth: SIDEBAR_WIDTH,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: GAP,
        padding: '10px 0',
        borderRadius: 0,
        background: railBackground,
        border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
        boxShadow: 'none',
        position: 'relative',
        height: '100%',
        overflow: 'hidden',
      }}
    >
      <div
        ref={containerRef}
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: GAP,
          width: '100%',
          flex: 1,
          padding: '8px 0',
          overflow: 'hidden',
        }}
      >
        {visibleItems.map((item) => {
          const active = activeView === item.id;
          return (
            <button
              key={item.id}
              type="button"
              title={item.label}
              onClick={() => {
                setOverflowOpen(false);
                onSelect(item.id);
              }}
              style={{
                width: ICON_SIZE,
                height: ICON_SIZE,
                background: active ? accent : theme === 'dark' ? '#0f3f5f' : '#ffffff',
                color: active ? '#fff' : textColor,
                border: active ? 'none' : `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
                borderRadius: 0,
                cursor: 'pointer',
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: ICON_FONT_SIZE,
                fontWeight: 700,
                transition: 'transform 120ms ease, background 120ms ease',
              }}
            >
              {item.icon}
            </button>
          );
        })}

        {hasOverflow && (
          <div style={{ width: '100%', display: 'flex', justifyContent: 'center' }}>
            <button
              type="button"
              onClick={() => setOverflowOpen((prev) => !prev)}
              style={{
                width: ICON_SIZE,
                height: ICON_SIZE,
                background: theme === 'dark' ? '#0f3f5f' : '#ffffff',
                color: textColor,
                border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
                borderRadius: 0,
                cursor: 'pointer',
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 18,
                fontWeight: 700,
              }}
            >
              ⋯
            </button>
          </div>
        )}
      </div>

      {hasOverflow && overflowOpen && (
        <div
          style={{
            position: 'absolute',
            left: SIDEBAR_WIDTH,
            bottom: 72,
            width: 220,
            maxHeight: 320,
            overflowY: 'auto',
            padding: 12,
            borderRadius: 18,
            background: theme === 'dark' ? '#0a2b45' : '#ffffff',
            border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
            boxShadow: '0 12px 30px rgba(22, 92, 146, 0.12)',
            zIndex: 30,
          }}
        >
          <strong style={{ display: 'block', marginBottom: 10, color: textColor }}>More</strong>
          {overflowItems.map((item) => (
            <button
              key={item.id}
              type="button"
              title={item.label}
              onClick={() => {
                setOverflowOpen(false);
                onSelect(item.id);
              }}
              style={{
                width: '100%',
                textAlign: 'left',
                padding: '10px 12px',
                borderRadius: 12,
                border: 'none',
                background: theme === 'dark' ? '#0a2b45' : '#f9fafb',
                color: textColor,
                cursor: 'pointer',
                marginBottom: 6,
              }}
            >
              {item.icon} <span style={{ marginLeft: 8, fontSize: 14 }}>{item.label}</span>
            </button>
          ))}
        </div>
      )}
    </aside>
  );
}
