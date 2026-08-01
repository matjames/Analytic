import { useMemo } from "react";
import { useLocation } from "react-router-dom";
import { initiatorSections } from "./sidebarConfig";

export function useInitiatorLayout(user) {
  const location = useLocation();

  // Build sidebar sections; for district role, insert a separate "Review" section after "Requests"
  const sidebarSections = useMemo(() => {
    if (user?.role !== "district") {
      return initiatorSections;
    }

    // Clone base sections to avoid mutating the shared config
    const sections = initiatorSections.map((section) => ({
      ...section,
      items: [...section.items],
    }));

    const reviewSection = {
      title: "Review",
      items: [
        {
          to: "/requests",
          label: "All Requests",
          iconClass: "bi bi-list",
        },
        {
          to: { pathname: "/requests", search: "?status=pending" },
          label: "Pending",
          iconClass: "bi bi-clock-history",
        },
        {
          to: { pathname: "/requests", search: "?status=approved" },
          label: "Approved",
          iconClass: "bi bi-check-circle",
        },
        {
          to: { pathname: "/requests", search: "?status=rejected" },
          label: "Rejected",
          iconClass: "bi bi-x-circle",
        },
      ],
    };

    const requestsIndex = sections.findIndex((s) => s.title === "Requests");
    if (requestsIndex >= 0) {
      sections.splice(requestsIndex + 1, 0, reviewSection);
    } else {
      sections.push(reviewSection);
    }

    return sections;
  }, [user]);

  const activeSection = useMemo(() => {
    const path = location.pathname;
    if (path.startsWith("/initiator/dashboard")) return "Dashboard";
    if (path.startsWith("/initiator/requests")) return "Requests";
    if (path.startsWith("/initiator/facilities")) return "Facilities";
    if (path.startsWith("/requests")) return "Review";
    return "Dashboard";
  }, [location.pathname]);

  const visibleSection = useMemo(() => {
    return sidebarSections.find((s) => s.title === activeSection) || sidebarSections[0];
  }, [activeSection, sidebarSections]);

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
    sidebarSections,
    activeSection,
    visibleSection,
    userDisplay,
  };
}
