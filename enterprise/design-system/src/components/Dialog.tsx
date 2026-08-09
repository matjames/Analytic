import React from 'react';
import clsx from 'clsx';

export interface DialogProps {
  open: boolean;
  onClose?: () => void;
  title?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  children?: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
}

export const Dialog: React.FC<DialogProps> = ({ open, onClose, title, size = 'md', children, footer, className }) => {
  if (!open) return null;

  const sizes = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-2xl',
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" role="dialog" aria-modal="true" aria-label={title}>
      <div className="flex min-h-full items-center justify-center p-4">
        <div
          className="fixed inset-0 bg-black/50"
          onClick={onClose}
          aria-hidden="true"
        />
        <div
          className={clsx(
            'relative bg-white rounded-xl shadow-xl w-full',
            sizes[size],
            className
          )}
        >
          {title && (
            <DialogHeader>
              <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
              {onClose && (
                <button
                  onClick={onClose}
                  className="text-gray-400 hover:text-gray-600 focus:outline-none"
                  aria-label="Close"
                >
                  ✕
                </button>
              )}
            </DialogHeader>
          )}
          {children && <DialogBody>{children}</DialogBody>}
          {footer && <DialogFooter>{footer}</DialogFooter>}
        </div>
      </div>
    </div>
  );
};

export const DialogHeader: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({ className, children, ...rest }) => (
  <div className={clsx('px-6 py-4 border-b border-gray-200 flex items-center justify-between', className)} {...rest}>
    {children}
  </div>
);

export const DialogBody: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({ className, children, ...rest }) => (
  <div className={clsx('px-6 py-4', className)} {...rest}>{children}</div>
);

export const DialogFooter: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({ className, children, ...rest }) => (
  <div className={clsx('px-6 py-4 border-t border-gray-200 flex justify-end gap-2', className)} {...rest}>
    {children}
  </div>
);

export default Dialog;