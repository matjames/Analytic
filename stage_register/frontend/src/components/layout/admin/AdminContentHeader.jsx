import React from "react";

export default function AdminContentHeader({ activeSection }) {
  const getTitle = () => {
    if (activeSection === "Review Requests" || activeSection === "Field Station Requests") {
      return "Field Station Requests";
    }
    if (activeSection === "Field Station List") {
      return "Field Station Registry";
    }
    if (activeSection === "Dashboard") {
      return "Dashboard";
    }
    return "Field Operations Registry";
  };

  const getDescription = () => {
    if (activeSection === "Review Requests") {
      return "Review and approve field station requests";
    }
    if (activeSection === "Field Station Requests") {
      return "Create, track, and review field station requests";
    }
    if (activeSection === "Field Station List") {
      return "Browse field survey stations and open details.";
    }
    return "Administration";
  };

  return (
    <div className="content-header">
      <h1 className="content-title mb-0">{getTitle()}</h1>
      <div className="text-muted small">{getDescription()}</div>
    </div>
  );
}
