import React from 'react';
import clsx from 'clsx';
import { Badge } from './Badge';

export interface NotificationAction {
  label: string;
  url?: string;
  onClick?: () => void;
  variant?: 'primary' | 'secondary' | 'danger';
}

export interface NotificationProps {
  id: string;
  title: string;
  body?: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  category?: string;
  sourceApp?: string;
  read?: boolean;
  archived?: boolean;
  deepLink?: string;
  actions?: NotificationAction[];
  attachments?: Array<{ name: string; url: string; mimeType?: string }>;
  createdAt?: string;
  onMarkRead?: (id: string) => void;
  onArchive?: (id: string) => void;
  onAction?: (id: string, action: NotificationAction) => void;
  className?: string;
}

export const Notification: React.FC<NotificationProps> = ({
  id,
  title,
  body,
  priority,
  category,
  sourceApp,
  read = false,
  deepLink,
  actions,
  attachments,
  createdAt,
  onMarkRead,
  onArchive,
  onAction,
  className,
}) => {
  const priorityStyles = {
    low: 'border-l-gray-300',
    medium: 'border-l-blue-500',
    high: 'border-l-yellow-500',
    urgent: 'border-l-red-600',
  };
  const priorityIcons = {
    low: '🔵',
    medium: '🟡',
    high: '🔶',
    urgent: '🔴',
  };

  return (
    <div
      className={clsx(
        'relative bg-white border border-gray-200 border-l-4 rounded-lg p-4 mb-2 transition-shadow',
        priorityStyles[priority],
        !read && 'bg-blue-50/50',
        className
      )}
    >
      <div className="flex items-start gap-3">
        <span className="text-lg" aria-hidden="true">{priorityIcons[priority]}</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-gray-900 truncate">{title}</span>
            {!read && <Badge variant="primary" size="sm">New</Badge>}
          </div>
          {body && <p className="mt-1 text-sm text-gray-600 line-clamp-2">{body}</p>}
          <div className="mt-1 flex items-center gap-2 text-xs text-gray-400">
            {sourceApp && <span className="capitalize">{sourceApp}</span>}
            {category && <span>•</span>}
            {category && <span className="capitalize">{category.replace('_', ' ')}</span>}
            {createdAt && <span>• {formatTimestamp(createdAt)}</span>}
          </div>

          {attachments && attachments.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-2">
              {attachments.map((att, i) => (
                <a
                  key={i}
                  href={att.url}
                  className="inline-flex items-center gap-1 text-xs text-blue-600 hover:text-blue-800"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  📎 {att.name}
                </a>
              ))}
            </div>
          )}

          {(actions || deepLink || !read || onArchive) && (
            <div className="mt-3 flex items-center gap-2">
              {deepLink && (
                <a
                  href={deepLink}
                  className="text-xs font-medium text-blue-600 hover:text-blue-800"
                >
                  View details →
                </a>
              )}
              {actions?.map((action, i) => (
                <button
                  key={i}
                  onClick={() => onAction?.(id, action)}
                  className={clsx(
                    'text-xs px-3 py-1.5 rounded-md font-medium transition-colors',
                    action.variant === 'danger' && 'bg-red-50 text-red-700 hover:bg-red-100',
                    action.variant === 'secondary' && 'bg-gray-100 text-gray-700 hover:bg-gray-200',
                    (!action.variant || action.variant === 'primary') && 'bg-blue-600 text-white hover:bg-blue-700'
                  )}
                >
                  {action.label}
                </button>
              ))}
              {!read && onMarkRead && (
                <button
                  onClick={() => onMarkRead(id)}
                  className="text-xs text-gray-400 hover:text-gray-600"
                >
                  Mark as read
                </button>
              )}
              {onArchive && (
                <button
                  onClick={() => onArchive(id)}
                  className="text-xs text-gray-400 hover:text-gray-600"
                >
                  Archive
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export const NotificationBell: React.FC<{ count?: number; onClick?: () => void; className?: string }> = ({
  count = 0,
  onClick,
  className,
}) => (
  <button
    onClick={onClick}
    className={clsx('relative p-2 rounded-lg hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500', className)}
    aria-label={`${count} unread notifications`}
  >
    <svg className="w-6 h-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
    </svg>
    {count > 0 && (
      <span className="absolute -top-1 -right-1 w-5 h-5 bg-red-600 text-white text-xs font-medium rounded-full flex items-center justify-center">
        {count > 99 ? '99+' : count}
      </span>
    )}
  </button>
);

function formatTimestamp(ts: string): string {
  const date = new Date(ts);
  if (isNaN(date.getTime())) return ts || '';
  const now = new Date();
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  return date.toLocaleDateString();
}

export default Notification;