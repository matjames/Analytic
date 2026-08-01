import React from "react";
import InitiatorHeader from "./initiator/InitiatorHeader";
import InitiatorRoutes from "./initiator/InitiatorRoutes";
import { useInitiatorLayout } from "./initiator/useInitiatorLayout";
import MainLayoutShell from "./MainLayoutShell";

export default function InitiatorLayout() {
  const user = JSON.parse(localStorage.getItem("user") || "null");
  const { activeSection, visibleSection, userDisplay } = useInitiatorLayout(user);

  const primaryNav = [
    {
      to: "/initiator/requests",
      label: "Requests",
      iconClass: "bi bi-file-earmark-text",
    },
    {
      to: "/initiator/facilities",
      label: "Facilities",
      iconClass: "bi bi-hospital",
    },
  ];

  return (
    <MainLayoutShell
      header={<InitiatorHeader user={user} userDisplay={userDisplay} />}
      breadcrumb={`Initiator / ${activeSection}`}
      primaryNav={primaryNav}
      secondarySections={[visibleSection]}
    >
      <InitiatorRoutes user={user} />
    </MainLayoutShell>
  );
}
