import React, { Fragment } from "react";
import { NavLink, useLocation } from "react-router-dom";
import "../../styles/adminLayout.css";

/**
 * Generic application shell with:
 * - optional header
 * - breadcrumb bar
 * - primary sidebar rail
 * - secondary sidebar with grouped links
 * - main content area
 *
 * This is meant to be reused by different roles (admin, initiator, reviewer, etc.)
 * by passing different navigation configs and children.
 */
export default function MainLayoutShell({
  header = null,
  breadcrumb = null,
  primaryNav = [],
  secondarySections = [],
  children,
}) {
  const location = useLocation();

  // Helper function to check if a NavLink should be active based on pathname and query params
  const isActive = (item, pathname, search) => {
    const itemPath = typeof item.to === "string" ? item.to : (item.to.pathname || item.to);
    const itemSearch = typeof item.to === "string" ? "" : (item.to.search || "");
    
    // First check if pathname matches
    if (pathname !== itemPath) {
      return false;
    }
    
    // If item has search params, check if they match
    if (itemSearch) {
      const urlParams = new URLSearchParams(search);
      const itemParams = new URLSearchParams(itemSearch);
      
      // Check if all query params in item exist and match in URL
      for (const [key, value] of itemParams.entries()) {
        if (urlParams.get(key) !== value) {
          return false;
        }
      }
      return true;
    } else {
      // If item has no search query, it's active only when URL also has no status param
      // This ensures "All Requests" is only active when there's no status filter
      const urlParams = new URLSearchParams(search);
      return !urlParams.has("status");
    }
  };

  return (
    <Fragment>
      {header}

      <div className="main-layout">
        {/* Primary sidebar rail */}
        <nav className="sidebar-primary">
          {primaryNav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              exact
              className="sidebar-item"
              activeClassName="active"
            >
              {item.iconClass ? (
                <i className={item.iconClass} />
              ) : item.icon ? (
                <span className="sidebar-icon">{item.icon}</span>
              ) : null}
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>

        {/* Secondary sidebar with grouped links */}
        <aside className="sidebar-secondary">
          {secondarySections.map((section) => (
            <div key={section.title}>
              <div className="sidebar-section-title">{section.title}</div>
              {section.items?.map((item) => {
                const active = isActive(item, location.pathname, location.search);
                return (
                  <NavLink
                    key={typeof item.to === "string" ? item.to : JSON.stringify(item.to)}
                    to={item.to}
                    exact
                    className={`sidebar-link ${active ? "active" : ""}`}
                    activeClassName="active"
                    isActive={(match, location) => {
                      return isActive(item, location.pathname, location.search);
                    }}
                  >
                    {item.iconClass ? (
                      <i className={item.iconClass} />
                    ) : item.icon ? (
                      <span className="sidebar-link-icon">{item.icon}</span>
                    ) : null}
                    <span>{item.label}</span>
                    {item.badge !== undefined && (
                      <span className={`sidebar-badge ${item.badgeClass || ""}`}>
                        {item.badge}
                      </span>
                    )}
                  </NavLink>
                );
              })}
            </div>
          ))}
        </aside>

        {/* Main content area */}
        <div className="main-content">{children}</div>
      </div>
    </Fragment>
  );
}


