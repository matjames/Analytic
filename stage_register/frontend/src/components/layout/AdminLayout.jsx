import React from "react";
import AdminHeader from "./admin/AdminHeader";
import AdminRoutes from "./admin/AdminRoutes";
import { useAdminLayout } from "./admin/useAdminLayout";
import { sidebarRail, reviewerRail } from "./admin/sidebarConfig";
import MainLayoutShell from "./MainLayoutShell";

export default function AdminLayout() {
  const user = JSON.parse(localStorage.getItem("user") || "null");
  const { isReviewer, sidebarSections, activeSection, visibleSection, userDisplay } = useAdminLayout(user);

  const primaryNav = sidebarSections.map((section) => {
    const firstItem = section.items[0];
    const label = section.title.split(" ")[0];
    const toPath = typeof firstItem.to === "string" ? firstItem.to : firstItem.to.pathname;
    const rail = (isReviewer ? reviewerRail : sidebarRail).find((r) => {
      // Match by path first, then by label
      if (toPath.startsWith(r.to)) return true;
      // For reviewer sections, match by title
      if (isReviewer && (section.title.includes(r.label) || r.label.includes(section.title.split(" ")[0]))) return true;
      // For full sections, try to match labels
      if (!isReviewer && (section.title.includes(r.label) || r.label.includes(label))) return true;
      return false;
    });
    
    return {
      to: toPath,
      label: label,
      iconClass:
        rail?.iconClass ||
        (label === "Requests"
          ? "bi bi-file-earmark-text"
          : label === "Review"
          ? "bi bi-clipboard-check"
          : label === "Facility"
          ? "bi bi-hospital"
          : label === "Dashboard"
          ? "bi bi-speedometer2"
          : undefined),
    };
  });

  return (
    <MainLayoutShell
      header={<AdminHeader user={user} userDisplay={userDisplay} />}
      breadcrumb=""
      primaryNav={primaryNav}
      secondarySections={[visibleSection]}
    >
      <AdminRoutes user={user} />
    </MainLayoutShell>
  );
}
