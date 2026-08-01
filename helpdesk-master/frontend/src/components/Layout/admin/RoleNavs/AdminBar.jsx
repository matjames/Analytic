/* eslint-disable jsx-a11y/anchor-is-valid */
import React from "react";
import { Link, useLocation } from "react-router-dom";

const AdminBar = () => {
  const location = useLocation()
  const isActive = (path) => location.pathname === path || location.pathname.startsWith(`${path}/`)

  const links = [
    { to: '/admin/dashboard', label: 'Dashboard', subtitle: 'Command center' },
    { to: '/admin/tickets', label: 'Work Queue', subtitle: 'Operations intake' },
    { to: '/admin/add/ticket', label: 'Create Work Item', subtitle: 'Capture new cases' },
    { to: '/admin/agents', label: 'Operations Agents', subtitle: 'Team roster' },
    { to: '/admin/knowledge-base', label: 'Knowledge Base', subtitle: 'Playbooks & guides' },
    { to: '/admin/videos/upload', label: 'Video Library', subtitle: 'Training assets' },
  ]

  return (
    <ul className="navbar-nav admin-nav">
      {links.map((item) => (
        <li className="nav-item" key={item.to}>
          <Link className={`nav-link ${isActive(item.to) ? 'active' : ''}`} to={item.to} role="button">
            <span className="nav-link-title">{item.label}</span>
            <small className="nav-link-subtitle">{item.subtitle}</small>
          </Link>
        </li>
      ))}
    </ul>
  );
};

export default AdminBar;
