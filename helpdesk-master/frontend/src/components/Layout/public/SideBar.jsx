import React from 'react'
import { NavLink, useLocation } from 'react-router-dom'

const SideBar = () => {
    const location = useLocation()
    const isActive = (path) => location.pathname === path || location.pathname.startsWith(`${path}/`)

    const groups = [
        {
            title: 'Operations',
            items: [
                { to: '/public', label: 'Overview', icon: 'fas fa-shield-alt' },
                { to: '/public/submit', label: 'Submit Work Item', icon: 'fas fa-clipboard-list' },
                { to: '/public/tickets', label: 'Work Queue', icon: 'fas fa-briefcase' },
                { to: '/public/kaban', label: 'Kanban Board', icon: 'fas fa-columns' },
            ],
        },
        {
            title: 'Knowledge',
            items: [
                { to: '/public/faq', label: 'Playbooks', icon: 'fas fa-book-medical' },
                { to: '/public/knowledgeBase', label: 'Knowledge Base', icon: 'fas fa-book-open' },
                { to: '/public/videos', label: 'Training', icon: 'fas fa-chalkboard-teacher' },
                { to: '/public/documentation', label: 'Documentation', icon: 'fas fa-file-alt' },
            ],
        },
        {
            title: 'Admin',
            items: [
                { to: '/admin/dashboard', label: 'Admin Workspace', icon: 'fas fa-chart-line' },
            ],
        },
    ]

    return (
        <aside className="sidebar">
            <div className="sidebar-brand">
                <div className="sidebar-brand-icon">
                    <i className="fas fa-project-diagram"></i>
                </div>
                <div>
                    <div className="sidebar-brand-title">StatGate Ops</div>
                    <div className="sidebar-brand-subtitle">Analytics operations</div>
                </div>
            </div>

            {groups.map((group) => (
                <div className="sidebar-group" key={group.title}>
                    <div className="sidebar-group-title">{group.title}</div>
                    {group.items.map((item) => (
                        <NavLink key={item.to} to={item.to} className={`sidebar-item ${isActive(item.to) ? 'active' : ''}`}>
                            <i className={item.icon}></i>
                            <span>{item.label}</span>
                        </NavLink>
                    ))}
                </div>
            ))}
        </aside>
    )
}

export default SideBar