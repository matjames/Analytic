import React from "react";
import { NavLink, useHistory, useLocation } from "react-router-dom";
import { sidebarRail, reviewerRail } from "./sidebarConfig";

export default function AdminSidebar({ sidebarSections, activeSection, visibleSection, isReviewer }) {
  const history = useHistory();
  const location = useLocation();

  // Helper to normalize paths (remove trailing slashes, ensure consistent format)
  const normalizePath = (path) => {
    if (!path) return "";
    return path.replace(/\/$/, ""); // Remove trailing slash
  };

  // Helper to check if a path matches an item's 'to' property
  const pathMatchesItem = (itemTo, currentPath, currentSearch) => {
    // Normalize paths for comparison
    const normalizedCurrentPath = normalizePath(currentPath);
    const fullPath = normalizedCurrentPath + (currentSearch || "");
    
    if (typeof itemTo === "string") {
      // Remove query string for path comparison
      const itemPath = normalizePath(itemTo.split("?")[0]);
      
      // Exact match (most common case)
      if (normalizedCurrentPath === itemPath) {
        return true;
      }
      
      // Check if current path starts with item path + "/" (for nested routes like /facilities/123)
      if (normalizedCurrentPath.startsWith(itemPath + "/")) {
        return true;
      }
      
      // Check query string match if item has query params
      if (itemTo.includes("?")) {
        const itemQuery = itemTo.split("?")[1];
        // Check exact match with query
        if (fullPath === itemTo) {
          return true;
        }
        // Check if path matches and search includes the query params
        if (normalizedCurrentPath === itemPath && currentSearch?.includes(itemQuery)) {
          return true;
        }
      }
      
      return false;
    }
    
    if (itemTo?.pathname) {
      const pathname = normalizePath(itemTo.pathname);
      const search = itemTo.search || "";
      
      // Exact pathname match
      if (normalizedCurrentPath === pathname) {
        // If search params are specified, check them
        if (search) {
          const searchParams = search.replace("?", "");
          return currentSearch?.includes(searchParams) || fullPath === pathname + search;
        }
        return true;
      }
      
      // Check if current path starts with item pathname + "/" (for nested routes)
      if (normalizedCurrentPath.startsWith(pathname + "/")) {
        return true;
      }
      
      return false;
    }
    
    return false;
  };

  return (
    <aside className="sidebar-left-container">
      <div className="sidebar-main">
        {sidebarSections.map((section) => {
          // Primary check: if any child item matches the current path - this works for ALL sections automatically
          const isActiveByChild = section.items.some((item) => {
            return pathMatchesItem(item.to, location.pathname, location.search);
          });
          
          // Secondary check: if section title matches activeSection (from useAdminLayout)
          const isActiveByTitle = section.title === activeSection;
          
          // Section is active if either any child item matches OR title matches
          // Child matching is primary because it's more reliable and works for all routes
          const isActive = isActiveByChild || isActiveByTitle;
          
          const firstItem = section.items[0];
          const firstWord = section.title.split(" ")[0];
          const railList = isReviewer ? reviewerRail : sidebarRail;
          
          // Find matching rail item - try exact label match first, then first word match, then by route
          const rail = railList.find((r) => r.label === section.title) ||
                      railList.find((r) => {
                        const railFirstWord = r.label.split(" ")[0];
                        return railFirstWord === firstWord;
                      }) ||
                      railList.find((r) => {
                        // Match by route if first item's 'to' matches rail's 'to'
                        const firstItemTo = typeof firstItem.to === "string" ? firstItem.to : firstItem.to?.pathname;
                        return r.to === firstItemTo;
                      });
          
          // Use rail label if found (this ensures "Org Units" displays fully), otherwise use section title
          const displayLabel = rail?.label || section.title;
          const iconClass = rail?.iconClass || "bi bi-circle";
          
          return (
            <button
              key={section.title}
              type="button"
              onClick={() => history.push(firstItem.to)}
              className={`sidebar-item border-0 bg-transparent ${isActive ? "active" : ""}`}
            >
              <span aria-hidden>
                <i className={iconClass}></i>
              </span>
              <div className="mt-1">{displayLabel}</div>
            </button>
          );
        })}
      </div>

      <div className="sidebar-secondary">
        <div className="sidebar-secondary-title">{visibleSection.title}</div>
        <nav className="nav flex-column">
          {visibleSection.items.map((item) => (
            <NavLink
              key={item.label}
              to={item.to}
              className="sidebar-sub-item"
              activeClassName="active"
            >
              <i aria-hidden>•</i>
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>
      </div>
    </aside>
  );
}
