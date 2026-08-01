import React, { useState, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import { getValidToken } from "../../utils/auth";

const Header = () => {
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const dropdownRef = useRef(null);
    const [isAuthed, setIsAuthed] = useState(() => !!getValidToken());

    // Keep isAuthed in sync when token changes or expires (getValidToken clears storage when expired)
    useEffect(() => {
        const checkAuth = () => {
            setIsAuthed(!!getValidToken());
        };

        // Periodically re-check token validity (handles silent expiry)
        const intervalId = setInterval(checkAuth, 60 * 1000);

        // React to storage changes (other tabs/windows)
        window.addEventListener("storage", checkAuth);

        // Initial check in case something changed before mount
        checkAuth();

        return () => {
            clearInterval(intervalId);
            window.removeEventListener("storage", checkAuth);
        };
    }, []);

    // Close dropdown when clicking outside
    useEffect(() => {
        const handleClickOutside = (event) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
                setDropdownOpen(false);
            }
        };

        if (dropdownOpen) {
            document.addEventListener('mousedown', handleClickOutside);
        }

        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
        };
    }, [dropdownOpen]);

    return (
        <header className="public-header">
            <div className="header-content">
                <div className="brand-section">
                    <img src="/statgate-logo.svg" alt="StatGate logo" className="brand-logo" />
                    <div className="brand-text">
                        <h1>StatGate</h1>
                        <p>Field Operations & Agent Workforce Registry</p>
                    </div>
                </div>

                <nav>
                    <ul className="nav-menu">
                        <li><Link to="/">Home</Link></li>
                        <li><Link to="/mfl">Field Operations Registry</Link></li>
                        {/* <li className="nav-item dropdown" ref={dropdownRef}>
                            <span 
                                className="nav-link dropdown-toggle" 
                                onClick={() => setDropdownOpen(!dropdownOpen)}
                                style={{ cursor: 'pointer', color: 'white' }}
                            >
                                Standard Downloads
                            </span>
                            <div className={`dropdown-menu ${dropdownOpen ? 'show' : ''}`}>
                                <Link to="/" className="dropdown-item" onClick={() => setDropdownOpen(false)}>
                                    <i className="bi bi-file-pdf-fill"></i>
                                    <div className="dropdown-text">
                                        <span className="dropdown-title">Master Facility List</span>
                                    </div>
                                </Link>
                                <Link to="/" className="dropdown-item" onClick={() => setDropdownOpen(false)}>
                                    <i className="bi bi-file-pdf-fill"></i>
                                    <div className="dropdown-text">
                                        <span className="dropdown-title">Facilities by Ownership</span>
                                    </div>
                                </Link>
                                <Link to="/" className="dropdown-item" onClick={() => setDropdownOpen(false)}>
                                    <i className="bi bi-file-pdf-fill"></i>
                                    <div className="dropdown-text">
                                        <span className="dropdown-title">Facilities by Level</span>
                                    </div>
                                </Link>
                                <Link to="/" className="dropdown-item" onClick={() => setDropdownOpen(false)}>
                                    <i className="bi bi-file-pdf-fill"></i>
                                    <div className="dropdown-text">
                                        <span className="dropdown-title">Facilities by Authority</span>
                                    </div>
                                </Link>
                                <Link to="/" className="dropdown-item" onClick={() => setDropdownOpen(false)}>
                                    <i className="bi bi-file-pdf-fill"></i>
                                    <div className="dropdown-text">
                                        <span className="dropdown-title">Facilities by Region</span>
                                    </div>
                                </Link>
                            </div>
                        </li> */}
                        <li><Link to="/sops">SOPs & Manuals</Link></li>
                        <li><Link to="/api-docs">API Docs</Link></li>
                    </ul>
                </nav>

                {!isAuthed && (
                    <div className="header-buttons">
                        <Link to="/register" className="btn btn-light btn-sm text-primary fw-semibold">
                            Create Account
                        </Link>
                        <Link to="/login" className="btn btn-outline-light btn-sm login-btn">
                            Login
                        </Link>
                    </div>
                )}
                {isAuthed && (
                    <Link to="/initiator/requests" className="btn btn-light btn-sm text-primary fw-semibold">
                        Field Station Requests
                    </Link>
                )}
            </div>
        </header>
    )
}

export default Header;
