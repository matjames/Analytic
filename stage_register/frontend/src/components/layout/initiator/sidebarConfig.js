export const initiatorSections = [
  {
    title: "Requests",
    items: [
      {
        to: "/initiator/requests",
        label: "Field Station Requests",
        iconClass: "bi bi-list-ul",
      },
      {
        to: "/initiator/requests/new",
        label: "New Field Station",
        iconClass: "bi bi-plus-circle",
      },
      {
        to: "/initiator/requests/update",
        label: "Update Field Station",
        iconClass: "bi bi-pencil-square",
      },
      {
        to: "/initiator/requests/deactivate",
        label: "Deactivate Field Station",
        iconClass: "bi bi-x-circle",
      },
      {
        to: "/initiator/requests/status",
        label: "Request Tracking",
        iconClass: "bi bi-clock-history",
      },
    ],
  },
  {
    title: "Field Stations",
    items: [
      {
        to: "/initiator/facilities",
        label: "My Field Stations",
        iconClass: "bi bi-hospital",
      },
    ],
  },
  {
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
  },
];
