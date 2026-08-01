import { useMemo, useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { fullSidebarSections, reviewerSidebarSections } from "./sidebarConfig";
import { RequestsApi } from "../../../helpers/api";

export function useAdminLayout(user) {
  const location = useLocation();
  const [requestStats, setRequestStats] = useState({
    total: 0,
    pending: 0,
    approved: 0,
    rejected: 0,
  });

  const isReviewer =
    user?.role === "district_approver" ||
    user?.role === "district" ||
    user?.role === "moh_clinical" ||
    user?.role === "moh_publisher";

  // Fetch request stats for sidebar badges
  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token || !user) return;

    async function loadStats() {
      try {
        const stats = await RequestsApi.getStats();
        setRequestStats({
          total: stats.total || 0,
          pending: stats.pending || 0,
          approved: stats.approved || 0,
          rejected: stats.rejected || 0,
        });
      } catch (e) {
        console.error("Failed to load request stats for sidebar:", e);
      }
    }

    // Load stats once on mount
    loadStats();
    
    // Only refresh stats when the page becomes visible (user returns to tab)
    // This prevents unnecessary API calls when the user is on another tab
    const handleVisibilityChange = () => {
      if (!document.hidden) {
        loadStats();
      }
    };
    
    document.addEventListener('visibilitychange', handleVisibilityChange);
    
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [user?.role]); // Only depend on role, not the entire user object

  const sidebarSections = useMemo(() => {
    const baseSections = !isReviewer ? fullSidebarSections : reviewerSidebarSections;

    // Deep clone so we can safely adjust routes per role
    let sections = baseSections.map((section) => ({
      ...section,
      items: section.items.map((item) => ({ ...item })),
    }));

    // For district role, remap reviewer URLs into /district/... namespace
    if (user?.role === "district") {
      sections = sections.map((section) => ({
        ...section,
        items: section.items.map((item) => {
          const next = { ...item };
          const to = item.to;

          if (typeof to === "string") {
            if (to === "/reviewer/dashboard") {
              next.to = "/district/dashboard";
            } else if (to === "/requests" || to.startsWith("/requests/")) {
              // Reviewer pages -> /district/review/requests...
              next.to = to.replace("/requests", "/district/review/requests");
            } else if (to === "/facilities" || to.startsWith("/facilities/")) {
              next.to = to.replace("/facilities", "/district/facilities");
            } else if (to === "/initiator/requests" || to.startsWith("/initiator/requests")) {
              // Initiator pages -> /district/requests...
              next.to = to.replace("/initiator/requests", "/district/requests");
            }
          } else if (to && typeof to === "object") {
            const pathname = to.pathname || "";
            if (pathname === "/requests" || pathname.startsWith("/requests/")) {
              next.to = { ...to, pathname: pathname.replace("/requests", "/district/review/requests") };
            } else if (pathname === "/facilities" || pathname.startsWith("/facilities/")) {
              next.to = { ...to, pathname: pathname.replace("/facilities", "/district/facilities") };
            }
          }

          return next;
        }),
      }));
    }

    // Update badge counts with real stats
    return sections.map((section) => {
      if (section.title === "Requests" || section.title === "Review Requests") {
        return {
          ...section,
          items: section.items.map((item) => {
            const toStr =
              typeof item.to === "string"
                ? item.to
                : item.to?.search || item.to?.pathname || "";

            // Update badges based on status filter
            if (toStr.includes("?status=pending") || toStr.includes("status=pending")) {
              return { ...item, badge: requestStats.pending };
            }
            if (toStr.includes("?status=approved") || toStr.includes("status=approved")) {
              return { ...item, badge: requestStats.approved };
            }
            if (toStr.includes("?status=rejected") || toStr.includes("status=rejected")) {
              return { ...item, badge: requestStats.rejected };
            }
            // Update "All Requests" badge (items that go to /requests or district-review requests without status filter)
            if (
              (item.label === "All Requests" || item.label === "Field Station Requests") &&
              (toStr === "/requests" ||
                toStr.startsWith("/requests") ||
                toStr === "/district/review/requests" ||
                toStr.startsWith("/district/review/requests"))
            ) {
              return { ...item, badge: requestStats.total };
            }
            return item;
          }),
        };
      }
      return section;
    });
  }, [isReviewer, requestStats, user?.role]);

  const activeSection = useMemo(() => {
    const path = location.pathname;
    if (path.startsWith("/district/dashboard") || path.startsWith("/reviewer/dashboard"))
      return "Dashboard";
    if (path.startsWith("/dashboard") && !isReviewer) return "Dashboard";
    if (path.startsWith("/direct-edits")) return "Direct Edits";
    if (path.startsWith("/district/facilities") || path.startsWith("/facilities"))
      return "Field Station List";
    if (path.startsWith("/district/requests")) return "Requests";
    if (path.startsWith("/district/review/requests") || path.startsWith("/requests"))
      return isReviewer ? "Review Requests" : "Requests";
    if (path.startsWith("/users")) return "Users";
    if (
      path.startsWith("/facility-levels") ||
      path.startsWith("/ownership-types") ||
      path.startsWith("/authority-types") ||
      path.startsWith("/facility-upload") ||
      path.startsWith("/documents") ||
      path.startsWith("/settings")
    )
      return "Settings";
    if (path.startsWith("/levels") || path.startsWith("/hierarchy-move") || path.startsWith("/hierarchy") || path.startsWith("/units")) return "Org Units";
    return sidebarSections[0]?.title || "Dashboard";
  }, [isReviewer, location.pathname, sidebarSections]);

  const visibleSection = sidebarSections.find((s) => s.title === activeSection) || sidebarSections[0];

  const userDisplay = useMemo(() => {
    const display = user?.username || user?.email || "User";
    const initials = display
      .split(" ")
      .map((p) => p[0])
      .join("")
      .slice(0, 2)
      .toUpperCase();
    const district = user?.district || user?.district_name || "District";
    return { display, initials, district };
  }, [user]);

  return {
    isReviewer,
    sidebarSections,
    activeSection,
    visibleSection,
    userDisplay,
  };
}
