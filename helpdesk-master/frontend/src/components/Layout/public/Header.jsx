import React from 'react'
import { Link, useLocation } from 'react-router-dom'

const Header = () => {
    const location = useLocation()
    
    const isActive = (path) => {
        return location.pathname === path
    }
    
    return (
        <nav className="navbar navbar-expand-lg top-navbar">
            <div className="container-fluid">
                <div className="d-flex align-items-center gap-3 me-4">
                    <img src="/statgate-logo.svg" alt="StatGate logo" className="helpdesk-brand-logo" />
                    <div>
                        <div className="helpdesk-brand-title">StatGate</div>
                        <div className="helpdesk-brand-subtitle">Operations Console</div>
                    </div>
                </div>
                <ul className="navbar-nav me-auto">
                    <li className="nav-item">
                        <Link className={`nav-link ${isActive('/public') ? 'active' : ''}`} to="/public">Operations Home</Link>
                    </li>
                    <li className="nav-item">
                        <Link className={`nav-link ${isActive('/public/submit') ? 'active' : ''}`} to="/public/submit">Operations Intake</Link>
                    </li>
                    <li className="nav-item">
                        <Link className={`nav-link ${isActive('/public/tickets') ? 'active' : ''}`} to="/public/tickets">Operations Queue</Link>
                    </li>
                    <li className="nav-item">
                        <Link className={`nav-link ${isActive('/public/faq') ? 'active' : ''}`} to="/public/faq">Playbooks</Link>
                    </li>
                    <li className="nav-item">
                        <Link className={`nav-link ${isActive('/public/knowledgeBase') ? 'active' : ''}`} to="/public/knowledgeBase">Knowledge Base</Link>
                    </li>
                    <li className="nav-item">
                        <Link className={`nav-link ${isActive('/public/videos') ? 'active' : ''}`} to="/public/videos">Training Library</Link>
                    </li>
                </ul>
                <Link to="/public" className="user-section text-decoration-none">
                    <i className="fas fa-rocket"></i>
                    <span>Operations Console</span>
                </Link>
            </div>
        </nav>
    )
}

export default Header