import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from "react-router-dom";
import 'bootstrap/dist/css/bootstrap.min.css';
import './styles/custom.css';
import App from "./routes";

// StatGate single-sign-on bootstrap: accept a Registry-issued token handed over
// by the launcher (?statgate_token canonical, ?registry_token legacy alias),
// persist it under the keys this app's API client reads, and strip it from the
// address bar so it is not retained in history or shared links.
(function bootstrapStatGateSSO() {
  if (typeof window === 'undefined') return;
  const sp = new URLSearchParams(window.location.search);
  const handed = sp.get('statgate_token') || sp.get('registry_token');
  if (!handed) return;
  ['token', 'registry_jwt'].forEach((k) => localStorage.setItem(k, handed));
  const u = new URL(window.location.href);
  u.searchParams.delete('statgate_token');
  u.searchParams.delete('registry_token');
  window.history.replaceState({}, document.title, `${u.pathname}${u.search}${u.hash}`);
})();

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(
  <BrowserRouter>
    <App />
  </BrowserRouter>
);
