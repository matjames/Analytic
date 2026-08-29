import React, { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { UsersApi } from "../../helpers/api";

export default function VerifyEmail() {
  const token = new URLSearchParams(useLocation().search).get("token") || "";
  const [state, setState] = useState({ loading: true, error: "", verified: false });
  useEffect(() => {
    if (!token) { setState({ loading: false, error: "Verification token is missing.", verified: false }); return; }
    UsersApi.verifyEmail(token)
      .then(() => setState({ loading: false, error: "", verified: true }))
      .catch((error) => setState({ loading: false, error: error.response?.data?.error || "Verification failed.", verified: false }));
  }, [token]);
  return <div className="container py-5" style={{ maxWidth: "32rem" }}><div className="card card-body"><h1 className="h3">Email verification</h1>{state.loading && <p>Verifying your email...</p>}{state.verified && <><div className="alert alert-success">Your email is verified. You can now sign in.</div><Link className="btn btn-primary" to="/login">Continue to login</Link></>}{state.error && <><div className="alert alert-danger">{state.error}</div><Link to="/login">Return to login</Link></>}</div></div>;
}
