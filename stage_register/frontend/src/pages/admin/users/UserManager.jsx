import React, { useEffect, useMemo, useState } from "react";
import Tree from "rc-tree";
import "rc-tree/assets/index.css";
import { UsersApi, UnitsApi, LevelsApi } from "../../../helpers/api";

const emptyForm = {
  role: "",
  first_name: "",
  last_name: "",
  email: "",
  username: "",
  password: "",
  organisation: "",
  phoneno: "",
  district_id: "",
};

export default function UserManager() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");
  const [form, setForm] = useState(emptyForm);
  const [editingId, setEditingId] = useState(null);
  const [treeData, setTreeData] = useState([]);
  const [levelsMap, setLevelsMap] = useState(new Map());
  const [showHierarchyModal, setShowHierarchyModal] = useState(false);
  const [showUserModal, setShowUserModal] = useState(false);
  const [showResetPasswordModal, setShowResetPasswordModal] = useState(false);
  const [resetPasswordUserId, setResetPasswordUserId] = useState(null);
  const [resetPasswordForm, setResetPasswordForm] = useState({ newPassword: "", confirmPassword: "" });
  const [resetPasswordError, setResetPasswordError] = useState("");
  const [selectedDistrictName, setSelectedDistrictName] = useState("");

  const sortedUsers = useMemo(
    () => [...users].sort((a, b) => a.id - b.id),
    [users]
  );

  async function loadUsers() {
    setLoading(true);
    setError("");
    try {
      const data = await UsersApi.list();
      setUsers(data);
    } catch (e) {
      setError(e.response?.data?.error || e.message);
    } finally {
      setLoading(false);
    }
  }

  async function loadLookups() {
    try {
      const [levels, treeRes] = await Promise.all([LevelsApi.list(), UnitsApi.tree()]);
      setLevelsMap(new Map(levels.map((l) => [l.id, l])));
      setTreeData(treeRes.tree || []);
    } catch (e) {
      console.error("Failed to load lookups:", e);
    }
  }

  const filteredTree = useMemo(() => {
    if (!treeData.length) return [];
    const filter = (nodes) =>
      nodes
        .map((n) => {
          const level = levelsMap.get(n.levelId);
          if (level && level.level_number > 3) return null;
          const kids = n.children ? filter(n.children) : [];
          return { ...n, children: kids };
        })
        .filter(Boolean);
    return filter(treeData);
  }, [treeData, levelsMap]);

  const rcTreeData = useMemo(() => {
    const mapNodes = (nodes) =>
      nodes.map((n) => ({
        key: String(n.id),
        title: n.name,
        children: n.children ? mapNodes(n.children) : [],
        raw: n,
      }));
    return filteredTree.length ? mapNodes(filteredTree) : [];
  }, [filteredTree]);

  // Create a map of mfl_uid to name for quick lookups
  const districtMap = useMemo(() => {
    const map = new Map();
    const traverse = (nodes) => {
      nodes.forEach((n) => {
        if (n.mfl_uid) {
          map.set(n.mfl_uid, n.name);
        }
        if (n.children) {
          traverse(n.children);
        }
      });
    };
    traverse(treeData);
    return map;
  }, [treeData]);

  // Helper function to get district name from UID
  const getDistrictName = (districtUid) => {
    if (!districtUid) return null;
    return districtMap.get(districtUid) || null;
  };

  // Expand top-level territories by default.
  const defaultExpandedKeys = useMemo(() => {
    return rcTreeData.map((node) => node.key);
  }, [rcTreeData]);

  function handleDistrictSelect(node) {
    if (node.raw && node.raw.mfl_uid) {
      setForm({ ...form, district_id: node.raw.mfl_uid });
      setSelectedDistrictName(node.raw.name || "");
      setShowHierarchyModal(false);
    }
  }

  useEffect(() => {
    loadUsers();
    loadLookups();
  }, []);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setForm((prev) => ({ ...prev, [name]: value }));
  };

  function handleOpenCreateModal() {
    setForm(emptyForm);
    setEditingId(null);
    setSelectedDistrictName("");
    setError("");
    setShowUserModal(true);
  }

  function handleCloseUserModal() {
    setShowUserModal(false);
    setForm(emptyForm);
    setEditingId(null);
    setSelectedDistrictName("");
    setError("");
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      if (editingId) {
        const payload = { ...form };
        if (!payload.password) delete payload.password; // don't overwrite when blank
        const updated = await UsersApi.update(editingId, payload);
        setUsers((prev) => prev.map((u) => (u.id === editingId ? updated : u)));
      } else {
        const { user, token } = await UsersApi.create(form);
        if (token) localStorage.setItem("token", token);
        setUsers((prev) => [...prev, user]);
      }
      handleCloseUserModal();
    } catch (e) {
      setError(e.response?.data?.error || e.message);
    } finally {
      setLoading(false);
    }
  }

  function startEdit(user) {
    setEditingId(user.id);
    const districtName = getDistrictName(user.district_id);
    setForm({
      role: user.role || "",
      first_name: user.first_name || "",
      last_name: user.last_name || "",
      email: user.email || "",
      username: user.username || "",
      password: "",
      organisation: user.organisation || "",
      phoneno: user.phoneno || "",
      district_id: user.district_id || "",
    });
    setSelectedDistrictName(districtName || "");
    setError("");
    setShowUserModal(true);
  }

  async function handleDelete(id) {
    if (!window.confirm("Delete this user?")) return;
    setLoading(true);
    setError("");
    try {
      await UsersApi.remove(id);
      setUsers((prev) => prev.filter((u) => u.id !== id));
      if (editingId === id) {
        handleCloseUserModal();
      }
    } catch (e) {
      setError(e.response?.data?.error || e.message);
    } finally {
      setLoading(false);
    }
  }

  function handleOpenResetPasswordModal(user) {
    setResetPasswordUserId(user.id);
    setResetPasswordForm({ newPassword: "", confirmPassword: "" });
    setResetPasswordError("");
    setShowResetPasswordModal(true);
  }

  function handleCloseResetPasswordModal() {
    setShowResetPasswordModal(false);
    setResetPasswordUserId(null);
    setResetPasswordForm({ newPassword: "", confirmPassword: "" });
    setResetPasswordError("");
  }

  async function handleResetPassword(e) {
    e.preventDefault();
    setResetPasswordError("");
    
    if (!resetPasswordForm.newPassword || resetPasswordForm.newPassword.trim() === "") {
      setResetPasswordError("Password is required");
      return;
    }

    if (resetPasswordForm.newPassword.length < 6) {
      setResetPasswordError("Password must be at least 6 characters long");
      return;
    }

    if (resetPasswordForm.newPassword !== resetPasswordForm.confirmPassword) {
      setResetPasswordError("Passwords do not match");
      return;
    }

    setLoading(true);
    try {
      await UsersApi.resetPassword(resetPasswordUserId, resetPasswordForm.newPassword);
      handleCloseResetPasswordModal();
      setSuccessMessage("Password reset successfully!");
      // Auto-hide success message after 5 seconds
      setTimeout(() => {
        setSuccessMessage("");
      }, 5000);
    } catch (e) {
      setResetPasswordError(e.response?.data?.error || e.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="content-wrapper" style={{ display: "block" }}>
        <div className="users-panel" style={{ width: "100%" }}>
          <div className="panel-header">
            <h2 className="panel-title">Users</h2>
            <div style={{ display: "flex", gap: "10px" }}>
              <button
                className="btn btn-primary"
                onClick={handleOpenCreateModal}
                disabled={loading}
              >
                <i className="bi bi-plus-circle"></i> Create User
              </button>
              <a
                href="/users/upload"
                className="btn btn-outline-primary"
                style={{ display: "flex", alignItems: "center", gap: "4px" }}
              >
                <i className="bi bi-upload"></i> Upload Users
              </a>
              <button
                className="btn btn-outline-secondary"
                onClick={loadUsers}
                disabled={loading}
              >
                <i className="bi bi-arrow-clockwise"></i> Refresh
              </button>
            </div>
          </div>

          {error && (
            <div className="alert alert-danger alert-dismissible fade show" role="alert" style={{ margin: "15px" }}>
              <strong>Error:</strong> {error}
              <button
                type="button"
                className="btn-close"
                onClick={() => setError("")}
                aria-label="Close"
              />
            </div>
          )}

          {successMessage && (
            <div className="alert alert-success alert-dismissible fade show" role="alert" style={{ margin: "15px" }}>
              <strong>Success:</strong> {successMessage}
              <button
                type="button"
                className="btn-close"
                onClick={() => setSuccessMessage("")}
                aria-label="Close"
              />
            </div>
          )}

          <div className="users-table-container" style={{ width: "100%", overflowX: "auto" }}>
            {loading && users.length === 0 ? (
              <div style={{ padding: "20px", textAlign: "center" }}>Loading users...</div>
            ) : (
              <table className="users-table" style={{ width: "100%", tableLayout: "auto" }}>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Username</th>
                    <th>First Name</th>
                    <th>Last Name</th>
                    <th>Email</th>
                    <th>Role</th>
                    <th>Organisation</th>
                    <th>Phone</th>
                    <th>District</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {sortedUsers.length === 0 ? (
                    <tr>
                      <td colSpan="8" style={{ textAlign: "center", padding: "20px" }}>
                        No users found
                      </td>
                    </tr>
                  ) : (
                    sortedUsers.map((user) => (
                      <tr key={user.id}>
                        <td>{user.id}</td>
                        <td>{user.username}</td>
                        <td>{user.first_name || "-"}</td>
                        <td>{user.last_name || "-"}</td>
                        <td>{user.email}</td>
                        <td>{user.role || "-"}</td>
                        <td>{user.organisation || "-"}</td>
                        <td>{user.phoneno || "-"}</td>
                        <td>{getDistrictName(user.district_id) || "-"}</td>
                        <td>
                          <div className="action-buttons">
                            <button
                              className="btn btn-sm btn-outline-primary"
                              onClick={() => startEdit(user)}
                              disabled={loading}
                            >
                              Edit
                            </button>
                            <button
                              className="btn btn-sm btn-outline-warning"
                              onClick={() => handleOpenResetPasswordModal(user)}
                              disabled={loading}
                              title="Reset Password"
                            >
                              Reset Password
                            </button>
                            <button
                              className="btn btn-sm btn-outline-danger"
                              onClick={() => handleDelete(user.id)}
                              disabled={loading}
                            >
                              Delete
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>

      {/* User Create/Edit Modal */}
      {showUserModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)" }}
          onClick={handleCloseUserModal}
        >
          <div
            className="modal-dialog modal-lg modal-dialog-scrollable"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">
                  {editingId ? "Edit User" : "Create User"}
                </h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={handleCloseUserModal}
                />
              </div>
              <form onSubmit={handleSubmit}>
                <div className="modal-body">
                  {error && (
                    <div className="alert alert-danger" role="alert">
                      {error}
                    </div>
                  )}

                  <div className="row">
                    <div className="col-md-6 mb-3">
                      <label className="form-label">
                        First Name
                      </label>
                      <input
                        type="text"
                        className="form-control"
                        name="first_name"
                        value={form.first_name}
                        onChange={handleChange}
                        placeholder="First name"
                      />
                    </div>

                    <div className="col-md-6 mb-3">
                      <label className="form-label">
                        Last Name
                      </label>
                      <input
                        type="text"
                        className="form-control"
                        name="last_name"
                        value={form.last_name}
                        onChange={handleChange}
                        placeholder="Last name"
                      />
                    </div>
                  </div>

                  <div className="row">
                    <div className="col-md-6 mb-3">
                      <label className="form-label">
                        Role <span className="text-danger">*</span>
                      </label>
                      <select
                        className="form-select"
                        name="role"
                        value={form.role}
                        onChange={handleChange}
                        required
                      >
                        <option value="">Select role...</option>
                        <option value="admin">Admin</option>
                        <option value="public">Public</option>
                        <option value="district">District</option>
                        <option value="moh_clinical">National Operations Reviewer</option>
                        <option value="moh_publisher">Registry Publisher</option>
                        <option value="integration">System Integration</option>
                      </select>
                    </div>

                    <div className="col-md-6 mb-3">
                      <label className="form-label">
                        Email <span className="text-danger">*</span>
                      </label>
                      <input
                        type="email"
                        className="form-control"
                        name="email"
                        value={form.email}
                        onChange={handleChange}
                        placeholder="Email"
                        required
                      />
                    </div>
                  </div>

                  <div className="row">
                    <div className="col-md-6 mb-3">
                      <label className="form-label">
                        Username <span className="text-danger">*</span>
                      </label>
                      <input
                        type="text"
                        className="form-control"
                        name="username"
                        value={form.username}
                        onChange={handleChange}
                        placeholder="Username"
                        required
                      />
                    </div>

                    <div className="col-md-6 mb-3">
                      <label className="form-label">
                        Password {!editingId && <span className="text-danger">*</span>}
                        {editingId && <span className="text-muted">(leave blank to keep current)</span>}
                      </label>
                      <input
                        type="password"
                        className="form-control"
                        name="password"
                        value={form.password}
                        onChange={handleChange}
                        placeholder="Password"
                        required={!editingId}
                      />
                    </div>
                  </div>

                  <div className="row">
                    <div className="col-md-6 mb-3">
                      <label className="form-label">Organisation</label>
                      <input
                        type="text"
                        className="form-control"
                        name="organisation"
                        value={form.organisation}
                        onChange={handleChange}
                        placeholder="Organisation"
                      />
                    </div>

                    <div className="col-md-6 mb-3">
                      <label className="form-label">Phone Number</label>
                      <input
                        type="tel"
                        className="form-control"
                        name="phoneno"
                        value={form.phoneno}
                        onChange={handleChange}
                        placeholder="Phone number"
                      />
                    </div>
                  </div>

                  <div className="row">
                    <div className="col-12 mb-3">
                      <label className="form-label">District (optional)</label>
                      <div className="input-group">
                        <input
                          type="text"
                          className="form-control"
                          value={selectedDistrictName}
                          placeholder="Select district from hierarchy (level 3)"
                          readOnly
                        />
                        <button
                          type="button"
                          className="btn btn-outline-secondary"
                          onClick={() => setShowHierarchyModal(true)}
                        >
                          Select
                        </button>
                      </div>
                      <div className="form-text">Only districts up to level 3 are shown</div>
                    </div>
                  </div>
                </div>
                <div className="modal-footer">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={handleCloseUserModal}
                    disabled={loading}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={loading}
                  >
                    {loading
                      ? "Saving..."
                      : editingId
                      ? "Update User"
                      : "Create User"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* District Hierarchy Modal */}
      {showHierarchyModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)" }}
          onClick={() => setShowHierarchyModal(false)}
        >
          <div
            className="modal-dialog modal-lg modal-dialog-scrollable"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">Select District</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => setShowHierarchyModal(false)}
                />
              </div>
              <div className="modal-body">
                {rcTreeData.length > 0 ? (
                  <Tree
                    treeData={rcTreeData}
                    defaultExpandedKeys={defaultExpandedKeys}
                    showIcon
                    selectable
                    onSelect={(keys, info) => handleDistrictSelect(info.node)}
                  />
                ) : (
                  <div>Loading hierarchy...</div>
                )}
              </div>
              <div className="modal-footer">
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowHierarchyModal(false)}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Reset Password Modal */}
      {showResetPasswordModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)" }}
          onClick={handleCloseResetPasswordModal}
        >
          <div
            className="modal-dialog modal-dialog-centered"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">Reset Password</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={handleCloseResetPasswordModal}
                />
              </div>
              <form onSubmit={handleResetPassword}>
                <div className="modal-body">
                  {resetPasswordError && (
                    <div className="alert alert-danger" role="alert">
                      {resetPasswordError}
                    </div>
                  )}

                  <div className="mb-3">
                    <label className="form-label">
                      New Password <span className="text-danger">*</span>
                    </label>
                    <input
                      type="password"
                      className="form-control"
                      value={resetPasswordForm.newPassword}
                      onChange={(e) =>
                        setResetPasswordForm({
                          ...resetPasswordForm,
                          newPassword: e.target.value,
                        })
                      }
                      placeholder="Enter new password"
                      required
                      minLength={6}
                    />
                    <div className="form-text">Password must be at least 6 characters long</div>
                  </div>

                  <div className="mb-3">
                    <label className="form-label">
                      Confirm Password <span className="text-danger">*</span>
                    </label>
                    <input
                      type="password"
                      className="form-control"
                      value={resetPasswordForm.confirmPassword}
                      onChange={(e) =>
                        setResetPasswordForm({
                          ...resetPasswordForm,
                          confirmPassword: e.target.value,
                        })
                      }
                      placeholder="Confirm new password"
                      required
                      minLength={6}
                    />
                  </div>
                </div>
                <div className="modal-footer">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={handleCloseResetPasswordModal}
                    disabled={loading}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={loading}
                  >
                    {loading ? "Resetting..." : "Reset Password"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
