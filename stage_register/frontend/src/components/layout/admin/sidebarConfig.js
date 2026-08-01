export const sidebarRail = [
  { to: "/dashboard", label: "Dashboard", iconClass: "bi bi-speedometer2" },
  { to: "/levels", label: "Org Units", iconClass: "bi bi-diagram-3" },
  { to: "/facilities", label: "Registry", iconClass: "bi bi-geo-alt" },
  { to: "/direct-edits", label: "Direct", iconClass: "bi bi-pencil-square" },
  { to: "/requests", label: "Requests", iconClass: "bi bi-file-earmark-text" },
  { to: "/users", label: "Users", iconClass: "bi bi-people" },
  { to: "/facility-levels", label: "Settings", iconClass: "bi bi-gear" },
];

export const reviewerRail = [
  { to: "/reviewer/dashboard", label: "Dashboard", iconClass: "bi bi-speedometer2" },
  { to: "/facilities", label: "Field Stations", iconClass: "bi bi-geo-alt" },
];

export const fullSidebarSections = [
  {
    title: "Dashboard",
    items: [
      { to: "/dashboard", label: "Overview", iconClass: "bi bi-speedometer2" },
    ],
  },
  {
    title: "Org Units",
    items: [
      { to: "/levels", label: "Define Levels", iconClass: "bi bi-list-nested" },
      { to: "/hierarchy-move", label: "Move Admin Hierarchy", iconClass: "bi bi-arrow-left-right" },
      { to: "/hierarchy", label: "Org Units", iconClass: "bi bi-diagram-2" },
      { to: "/units", label: "Uploads", iconClass: "bi bi-building" },
    ],
  },
  {
    title: "Direct Edits",
    items: [
      { to: "/direct-edits/add", label: "Direct Addition", iconClass: "bi bi-plus-circle" },
      { to: "/direct-edits/update", label: "Direct Update", iconClass: "bi bi-pencil-square" },
      { to: "/direct-edits/deactivate", label: "Direct Deactivation", iconClass: "bi bi-x-circle" },
    ],
  },
  {
    title: "Requests",
    items: [
      { to: "/requests", label: "Field Station Requests", iconClass: "bi bi-list-ul" },
      { to: "/requests?status=pending", label: "Pending", iconClass: "bi bi-clock-history", badge: 8, badgeClass: "sidebar-badge-warning" },
      { to: "/requests?status=approved", label: "Approved", iconClass: "bi bi-check-circle", badge: 5, badgeClass: "sidebar-badge-success" },
      { to: "/requests?status=rejected", label: "Rejected", iconClass: "bi bi-x-circle", badge: 2, badgeClass: "sidebar-badge-danger" },
      { to: "/requests", label: "Request Tracking", iconClass: "bi bi-clock-history" },
    ],
  },
  {
    title: "Field Operations Registry",
    items: [
      { to: "/facilities", label: "All Field Stations", iconClass: "bi bi-geo-alt" },
      { to: { pathname: "/facilities", search: "?status=functional" }, label: "Functional", iconClass: "bi bi-check-circle" },
      { to: { pathname: "/facilities", search: "?status=non-functional" }, label: "Non Functional", iconClass: "bi bi-x-circle" },
      { to: { pathname: "/facilities", search: "?reporting=yes" }, label: "Reporting", iconClass: "bi bi-file-earmark-text" },
      { to: { pathname: "/facilities", search: "?reporting=no" }, label: "Not Reporting", iconClass: "bi bi-file-x" },
    ],
  },
  {
    title: "Users",
    items: [
      { to: "/users", label: "All Users", iconClass: "bi bi-people" },
      { to: "/users/upload", label: "Upload Users", iconClass: "bi bi-upload" },
      { to: "/roles", label: "User Roles", iconClass: "bi bi-shield-lock" },
      { to: "/permissions", label: "Permissions", iconClass: "bi bi-key" }
    ],
  },
  {
    title: "Settings",
    items: [
      { to: "/facility-levels", label: "Station Tiers", iconClass: "bi bi-layers" },
      { to: "/ownership-types", label: "Operating Models", iconClass: "bi bi-diagram-2" },
      { to: "/authority-types", label: "Managing Organisations", iconClass: "bi bi-shield" },
      { to: "/facility-upload", label: "Upload Field Stations", iconClass: "bi bi-upload" },
      { to: "/documents", label: "Public Documents", iconClass: "bi bi-file-pdf" },
    ],
  },
];

export const reviewerSidebarSections = [
  {
    title: "Dashboard",
    items: [{ to: "/reviewer/dashboard", label: "Overview" }]
  },
  {
    title: "Requests",
    items: [
      { to: "/initiator/requests", label: "Field Station Requests", iconClass: "bi bi-list-ul" },
      { to: "/initiator/requests/new", label: "New Field Station", iconClass: "bi bi-plus-circle" },
      { to: "/initiator/requests/update", label: "Update Field Station", iconClass: "bi bi-pencil-square" },
      { to: "/initiator/requests/deactivate", label: "Deactivate Field Station", iconClass: "bi bi-x-circle" },
      { to: "/initiator/requests/status", label: "Request Tracking", iconClass: "bi bi-clock-history" },
    ]
  },
  {
    title: "Review Requests",
    items: [
      { to: "/requests", label: "All Requests", iconClass: "bi bi-list", badge: 15 },
      { to: { pathname: "/requests", search: "?status=pending" }, label: "Pending", iconClass: "bi bi-clock-history", badge: 8, badgeClass: "sidebar-badge-warning" },
      { to: { pathname: "/requests", search: "?status=approved" }, label: "Approved", iconClass: "bi bi-check-circle", badge: 5, badgeClass: "sidebar-badge-success" },
      { to: { pathname: "/requests", search: "?status=rejected" }, label: "Rejected", iconClass: "bi bi-x-circle", badge: 2, badgeClass: "sidebar-badge-danger" },
    ]
  },
  {
    title: "Field Station Registry",
    items: [
      { to: "/facilities", label: "All Field Stations", iconClass: "bi bi-geo-alt" },
      { to: { pathname: "/facilities", search: "?status=functional" }, label: "Functional", iconClass: "bi bi-check-circle" },
      { to: { pathname: "/facilities", search: "?status=non-functional" }, label: "Non Functional", iconClass: "bi bi-x-circle" },
      { to: { pathname: "/facilities", search: "?reporting=yes" }, label: "Reporting", iconClass: "bi bi-file-earmark-text" },
      { to: { pathname: "/facilities", search: "?reporting=no" }, label: "Not Reporting", iconClass: "bi bi-file-x" },
    ]
  },
];
