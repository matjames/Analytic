import React from 'react';
import clsx from 'clsx';

export interface LayoutProps extends React.HTMLAttributes<HTMLDivElement> {
  sidebar?: React.ReactNode;
  header?: React.ReactNode;
  sidebarWidth?: string;
}

export const Layout: React.FC<LayoutProps> = ({
  sidebar,
  header,
  sidebarWidth = 'w-64',
  className,
  children,
  ...rest
}) => (
  <div className={clsx('flex min-h-screen bg-gray-50', className)} {...rest}>
    {sidebar && (
      <aside className={clsx('fixed inset-y-0 left-0 bg-white border-r border-gray-200', sidebarWidth)}>
        {sidebar}
      </aside>
    )}
    <div className={clsx('flex-1 flex flex-col', sidebar ? 'ml-64' : '')}>
      {header && <Header>{header}</Header>}
      <Content>{children}</Content>
    </div>
  </div>
);

export const Header: React.FC<React.HTMLAttributes<HTMLElement>> = ({ className, children, ...rest }) => (
  <header className={clsx('bg-white border-b border-gray-200 px-6 py-3 flex items-center justify-between', className)} {...rest}>
    {children}
  </header>
);

export const Sidebar: React.FC<React.HTMLAttributes<HTMLElement>> = ({ className, children, ...rest }) => (
  <nav className={clsx('h-full p-4', className)} {...rest}>{children}</nav>
);

export const Content: React.FC<React.HTMLAttributes<HTMLElement>> = ({ className, children, ...rest }) => (
  <main className={clsx('flex-1 p-6', className)} {...rest}>{children}</main>
);

export default Layout;