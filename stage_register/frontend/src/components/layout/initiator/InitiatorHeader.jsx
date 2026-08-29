import React, { useState, useRef, useEffect } from "react";
import { UsersApi } from "../../../helpers/api/users";
import { useHistory } from "react-router-dom";
import launcherApps from "../../../config/launcherApps";
import useOrganisationBranding from "../../../hooks/useOrganisationBranding";

export default function InitiatorHeader({ user, userDisplay }) {
  const history = useHistory();
  const branding = useOrganisationBranding();
  const [showDropdown, setShowDropdown] = useState(false);
  const [showLauncherMenu, setShowLauncherMenu] = useState(false);
  const dropdownRef = useRef(null);
  const launcherRef = useRef(null);

  const handleLogout = async () => {
    try { await UsersApi.logout(); } catch { /* local logout still clears the session */ }
    localStorage.removeItem("token");
    localStorage.removeItem("refresh_token");
    localStorage.removeItem("user");
    window.location.href = "/";
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setShowDropdown(false);
      }
      if (launcherRef.current && !launcherRef.current.contains(event.target)) {
        setShowLauncherMenu(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  if (!user) return null;

  // Format role for display
  const formatRole = (role) => {
    if (!role) return "User";
    return role
      .split("_")
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(" ");
  };

  return (
    <div
      className="header"
      style={{
        "--primary-color": branding.primary_color,
        "--primary-dark": branding.secondary_color,
      }}
    >
      <div className="header-content">
        <div
          className="header-left"
          onClick={() => history.push("/")}
          style={{ cursor: "pointer" }}
        >
          <img
            src={branding.logo_url || "/statgate-logo.svg"}
            alt={`${branding.display_name} logo`}
            className="logo-badge"
          />
          <div className="header-title">
            <div className="header-title-line">
              <span className="header-country">{branding.display_name}</span>
            </div>
            <div className="header-title-line">
              <span className="header-ministry">FIELD OPERATIONS</span>
            </div>
          </div>
          <div className="header-subtitle">
            <span className="header-registry">Agent Workforce Registry</span>
          </div>
        </div>
        <div className="header-right">
          <div className="launcher-dropdown" ref={launcherRef}>
            <button
              className="launcher-toggle"
              onClick={() => setShowLauncherMenu(!showLauncherMenu)}
              aria-label="Open app launcher"
              aria-expanded={showLauncherMenu}
            >
              <i className="bi bi-grid-3x3-gap-fill"></i>
            </button>
            {showLauncherMenu && (
              <div className="launcher-menu">
                {launcherApps.map((app) => (
                  <a key={app.name} href={app.url} className="launcher-item">
                    <i className={app.icon}></i>
                    <div>
                      <div className="launcher-item-title">{app.name}</div>
                      <div className="launcher-item-desc">{app.description}</div>
                    </div>
                  </a>
                ))}
              </div>
            )}
          </div>
          <div className="user-profile-dropdown" ref={dropdownRef}>
            <button
              className="user-avatar-btn"
              onClick={() => setShowDropdown(!showDropdown)}
              aria-label="User menu"
              aria-expanded={showDropdown}
            >
              <div className="user-avatar-circle">
                {userDisplay.initials}
              </div>
              <div className="user-avatar-text">
                <span className="user-avatar-name">{userDisplay.display}</span>
              </div>
            </button>
            {showDropdown && (
              <div className="user-dropdown-menu">
                <div className="dropdown-header">
                  <div className="dropdown-avatar">
                    {userDisplay.initials}
                  </div>
                  <div className="dropdown-user-info">
                    <div className="dropdown-user-name">{userDisplay.display}</div>
                    <div className="dropdown-user-email">{user.email || user.username}</div>
                  </div>
                </div>
                <div className="dropdown-divider"></div>
                <div className="dropdown-content">
                  <div className="dropdown-item">
                    <i className="bi bi-person-badge"></i>
                    <div className="dropdown-item-content">
                      <div className="dropdown-item-label">Role</div>
                      <div className="dropdown-item-value">{formatRole(user.role)}</div>
                    </div>
                  </div>
                  {userDisplay.district && userDisplay.district !== "District" && (
                    <div className="dropdown-item">
                      <i className="bi bi-geo-alt-fill"></i>
                      <div className="dropdown-item-content">
                        <div className="dropdown-item-label">District</div>
                        <div className="dropdown-item-value">{userDisplay.district}</div>
                      </div>
                    </div>
                  )}
                </div>
                <div className="dropdown-divider"></div>
                <button className="dropdown-logout-btn" onClick={handleLogout}>
                  <i className="bi bi-box-arrow-right"></i>
                  <span>Logout</span>
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
