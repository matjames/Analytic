import React, { useState, useRef, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import Launcher from '../../Launcher/Launcher'

const launcherApps = [
    { name: 'Dashboard', url: 'http://localhost:5000', icon: 'fa fa-home', external: true },
    { name: 'Dataset Catalog', url: 'http://localhost:5000/datasets', icon: 'fa fa-table', external: true },
    { name: 'Notebook', url: 'http://localhost:5000/notebook', icon: 'fa fa-book', external: true },
    { name: 'Semantic Registry', url: 'http://localhost:5000/semantic', icon: 'fa fa-brain', external: true },
    { name: 'ABAC Security', url: 'http://localhost:5000/abac', icon: 'fa fa-shield-alt', external: true },
    { name: 'Executive Centre', url: 'http://localhost:5000/executive', icon: 'fa fa-university', external: true },
    { name: 'System Launcher', url: 'http://localhost:3002', icon: 'bi bi-grid-3x3-gap-fill', external: true },
    { name: 'Register Portal', url: 'http://localhost:3000', icon: 'fa fa-edit', external: true },
]

const Header = () => {
    const location = useLocation()
    const [showLauncher, setShowLauncher] = useState(false)
    const launcherRef = useRef(null)

    const isActive = (path) => {
        return location.pathname === path
    }

    useEffect(() => {
        const handleClickOutside = (event) => {
            if (launcherRef.current && !launcherRef.current.contains(event.target)) {
                setShowLauncher(false)
            }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])
    
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
                {/* Use shared Launcher component */}
                <div>
                  {/* Launcher component injected here */}
                  <Launcher />
                </div>
            </div>
        </nav>
    )
}

export default Header