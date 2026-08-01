import React, { useMemo, useState, useEffect } from "react";
import { Redirect, Route, Switch } from "react-router-dom";
import AdminLayout from "../components/layout/AdminLayout";
import InitiatorLayout from "../components/layout/InitiatorLayout";
import LandingPage from "../pages/public/LandingPage";
import Login from "../pages/auth/Login";
import PublicMasterFacilityList from "../pages/public/PublicMasterFacilityList";
import PublicFacilityDetail from "../pages/public/PublicFacilityDetail";
import SopsManuals from "../pages/public/SopsManuals";
import ApiDocs from "../pages/public/ApiDocs";
import Register from "../pages/auth/Register";
import { getRoleRoute } from "../utils/roleRoutes";
import { getValidToken } from "../utils/auth";

export default function App() {
  const [, setAuthCheck] = useState(0);
  useEffect(() => {
    const handler = () => setAuthCheck((n) => n + 1);
    window.addEventListener("auth-expired", handler);
    return () => window.removeEventListener("auth-expired", handler);
  }, []);

  const token = getValidToken();
  const isAuthenticated = !!token;

  // Use useMemo to prevent recalculation on every render
  const defaultRoute = useMemo(() => {
    if (!isAuthenticated) return "/login";
    try {
      const user = JSON.parse(localStorage.getItem("user") || "null");
      return user ? getRoleRoute(user.role) : "/login";
    } catch {
      return "/login";
    }
  }, [isAuthenticated]);

  return (
    <Switch>
      <Route exact path="/" component={LandingPage} />
      <Route exact path="/mfl" component={PublicMasterFacilityList} />
      <Route exact path="/mfl/:id" component={PublicFacilityDetail} />
      <Route exact path="/sops" component={SopsManuals} />
      <Route exact path="/api-docs" component={ApiDocs} />
      <Route exact path="/register" component={Register} />
      <Route
        path="/login"
        render={() => (isAuthenticated ? <Redirect to={defaultRoute} /> : <Login />)}
      />
      <Route
        path="/initiator"
        render={() => {
          if (!isAuthenticated) return <Redirect to="/login" />;
          const user = JSON.parse(localStorage.getItem("user") || "null");
          if (
            user?.role === "public" ||
            user?.role === "district_initiator" ||
            user?.role === "district"
          ) {
            return <InitiatorLayout />;
          }
          return <Redirect to={getRoleRoute(user?.role)} />;
        }}
      />
      <Route
        path="/"
        render={() => (isAuthenticated ? <AdminLayout /> : <Redirect to="/login" />)}
      />
    </Switch>
  );
}