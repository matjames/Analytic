import React, { useEffect, useState } from "react";
import axios from "axios";
import { getValidToken } from "../../utils/auth";

const enterpriseBase = process.env.REACT_APP_ENTERPRISE_API_URL || "http://localhost:8096/api";

export default function WorkspaceSwitcher() {
  const [workspaces, setWorkspaces] = useState([]);
  const [selected, setSelected] = useState(() => localStorage.getItem("workspace_id") || "");

  useEffect(() => {
    const token = getValidToken();
    if (!token) return undefined;
    axios.get(`${enterpriseBase}/workspaces`, { headers: { Authorization: `Bearer ${token}` } })
      .then(({ data }) => {
        const items = data.workspaces || [];
        setWorkspaces(items);
        if (!items.some((item) => item.id === selected)) {
          const next = items[0]?.id || "";
          setSelected(next);
          if (next) localStorage.setItem("workspace_id", next);
          else localStorage.removeItem("workspace_id");
        }
      })
      .catch(() => setWorkspaces([]));
  }, [selected]);

  if (workspaces.length === 0) return null;

  const changeWorkspace = (event) => {
    const workspaceID = event.target.value;
    setSelected(workspaceID);
    localStorage.setItem("workspace_id", workspaceID);
    window.dispatchEvent(new CustomEvent("workspace-changed", { detail: workspaceID }));
  };

  return (
    <div className="workspace-context-bar" role="region" aria-label="Workspace context">
      <span className="workspace-context-label">Workspace</span>
      <select value={selected} onChange={changeWorkspace} aria-label="Select workspace">
        {workspaces.map((workspace) => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}
      </select>
    </div>
  );
}
