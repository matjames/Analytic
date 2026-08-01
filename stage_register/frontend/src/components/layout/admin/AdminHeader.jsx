import React, { useState, useRef, useEffect } from "react";
import { useHistory } from "react-router-dom";

export default function AdminHeader({ user, userDisplay }) {
  const history = useHistory();
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef(null);

  const handleLogout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    window.location.href = "/";
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setShowDropdown(false);
      }
    }

    if (showDropdown) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showDropdown]);

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
    <div className="header">
      <div className="header-content">
        <div className="header-left" onClick={() => history.push("/")} style={{ cursor: "pointer" }}>
          <img
            src={require("./logo.png")}
            alt="StatGate logo"
            className="logo-badge"
          />
          <div className="header-title">
            <div className="header-title-line">
              <span className="header-country">STATGATE</span>
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
