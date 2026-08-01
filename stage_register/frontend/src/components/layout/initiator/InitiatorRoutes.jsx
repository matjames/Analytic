import React from "react";
import { Switch, Route, Redirect } from "react-router-dom";
import Dashboard from "../../../pages/initiator/Dashboard";
import Requests from "../../../pages/initiator/Requests";
import RequestCreate from "../../../pages/initiator/RequestCreate";
import RequestUpdate from "../../../pages/initiator/RequestUpdate";
import RequestDeactivate from "../../../pages/initiator/RequestDeactivate";
import FacilityList from "../../../pages/initiator/FacilityList";
import FacilityDetail from "../../../pages/initiator/FacilityDetail";
import RequestStatus from "../../../pages/initiator/RequestStatus";
import { getRoleRoute } from "../../../utils/roleRoutes";

export default function InitiatorRoutes({ user }) {
  return (
    <Switch>
      <Route exact path="/initiator/dashboard" component={Dashboard} />
      <Route exact path="/initiator/requests" component={Requests} />
      <Route exact path="/initiator/requests/new" component={RequestCreate} />
      <Route exact path="/initiator/requests/update" component={RequestUpdate} />
      <Route exact path="/initiator/requests/deactivate" component={RequestDeactivate} />
      <Route exact path="/initiator/requests/status" component={RequestStatus} />
      <Route exact path="/initiator/facilities/:id" component={FacilityDetail} />
      <Route exact path="/initiator/facilities" component={FacilityList} />
      <Redirect to={getRoleRoute(user?.role)} />
    </Switch>
  );
}
