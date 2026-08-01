import React from "react";
import { Switch, Route, Redirect } from "react-router-dom";
import LevelsManager from "../../../pages/admin/hierarchy/LevelsManager";
import UnitsManager from "../../../pages/admin/organization/UnitsManager";
import HierarchyTree from "../../../pages/admin/hierarchy/HierarchyTree";
import HierarchyMove from "../../../pages/admin/hierarchy/HierarchyMove";
import FacilityLevelManager from "../../../pages/admin/facility/FacilityLevelManager";
import OwnershipTypeManager from "../../../pages/admin/facility/OwnershipTypeManager";
import AuthorityTypeManager from "../../../pages/admin/facility/AuthorityTypeManager";
import FacilityList from "../../../pages/admin/facility/FacilityList";
import FacilityFormPage from "../../../pages/admin/facility/FacilityFormPage";
import FacilityDetail from "../../../pages/admin/facility/FacilityDetail";
import FacilityManager from "../../../pages/admin/direct-edits/FacilityManager";
import UserManager from "../../../pages/admin/users/UserManager";
import UserUpload from "../../../pages/admin/users/UserUpload";
import RequestManager from "../../../pages/admin/requests/RequestManager";
import RequestDetailPage from "../../../pages/admin/requests/RequestDetailPage";
import ReviewerDashboard from "../../../pages/admin/dashboard/ReviewerDashboard";
import AdminDashboard from "../../../pages/admin/dashboard/AdminDashboard";
import DirectUpdates from "../../../pages/admin/direct-edits/DirectUpdates";
import DirectDeactivate from "../../../pages/admin/direct-edits/DirectDeactivate";
import Settings from "../../../pages/admin/settings";
import DocumentManager from "../../../pages/admin/settings/DocumentManager";
import FacilityUpload from "../../../pages/admin/settings/FacilityUpload";

// District initiator/review pages reused under /district/*
import InitiatorRequests from "../../../pages/initiator/Requests";
import RequestCreate from "../../../pages/initiator/RequestCreate";
import RequestUpdate from "../../../pages/initiator/RequestUpdate";
import RequestDeactivate from "../../../pages/initiator/RequestDeactivate";
import RequestStatus from "../../../pages/initiator/RequestStatus";
import InitiatorFacilityList from "../../../pages/initiator/FacilityList";
import InitiatorFacilityDetail from "../../../pages/initiator/FacilityDetail";

import { getRoleRoute } from "../../../utils/roleRoutes";

export default function AdminRoutes({ user }) {
  return (
    <Switch>
      {/* District-scoped aliases */}
      <Route path="/district/dashboard" component={ReviewerDashboard} />
      {/* Initiation-style district requests */}
      <Route exact path="/district/requests" component={InitiatorRequests} />
      <Route exact path="/district/requests/new" component={RequestCreate} />
      <Route exact path="/district/requests/update" component={RequestUpdate} />
      <Route exact path="/district/requests/deactivate" component={RequestDeactivate} />
      <Route exact path="/district/requests/status" component={RequestStatus} />
      <Route exact path="/district/facilities/:id" component={InitiatorFacilityDetail} />
      <Route exact path="/district/facilities" component={InitiatorFacilityList} />

      {/* District review aliases */}
      <Route exact path="/district/review/requests/:id" component={RequestDetailPage} />
      <Route exact path="/district/review/requests" component={RequestManager} />

      {/* Existing reviewer/admin routes remain for other roles */}
      <Route path="/reviewer/dashboard" component={ReviewerDashboard} />
      <Route path="/dashboard" component={AdminDashboard} />
      <Route path="/direct-edits/add" component={FacilityManager} />
      <Route path="/direct-edits/update" component={DirectUpdates} />
      <Route path="/direct-edits/deactivate" component={DirectDeactivate} />
      <Route exact path="/direct-edits" component={FacilityManager} />
      <Route path="/levels" component={LevelsManager} />
      <Route path="/units" component={UnitsManager} />
      <Route path="/hierarchy" component={HierarchyTree} />
      <Route path="/hierarchy-move" component={HierarchyMove} />
      <Route path="/facility-levels" component={FacilityLevelManager} />
      <Route path="/ownership-types" component={OwnershipTypeManager} />
      <Route path="/authority-types" component={AuthorityTypeManager} />
      <Route path="/facility-upload" component={FacilityUpload} />
      <Route path="/documents" component={DocumentManager} />
      <Route path="/facilities/new" component={FacilityFormPage} />
      <Route path="/facilities/:id/edit" component={FacilityFormPage} />
      <Route path="/facilities/:id" component={FacilityDetail} />
      <Route path="/facilities" component={FacilityList} />
      <Route path="/requests/:id" component={RequestDetailPage} />
      <Route path="/requests" component={RequestManager} />
      <Route path="/users/upload" component={UserUpload} />
      <Route path="/users" component={UserManager} />
      <Route path="/settings" component={Settings} />
      <Redirect to={getRoleRoute(user?.role)} />
    </Switch>
  );
}
