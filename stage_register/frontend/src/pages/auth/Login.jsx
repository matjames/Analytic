import React, { useState } from "react";
import { Link } from "react-router-dom";
import { UsersApi } from "../../helpers/api";
import { getRoleRoute } from "../../utils/roleRoutes";
import "./styles.css";

export default function Login() {
  const [form, setForm] = useState({ emailOrUsername: "", password: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [rememberMe, setRememberMe] = useState(false);
  const [showChangePasswordModal, setShowChangePasswordModal] = useState(false);
  const [changePasswordForm, setChangePasswordForm] = useState({
    emailOrUsername: "",
    oldPassword: "",
    newPassword: "",
  });
  const [changePasswordError, setChangePasswordError] = useState("");
  const [changePasswordSuccess, setChangePasswordSuccess] = useState("");
  const [changePasswordLoading, setChangePasswordLoading] = useState(false);
  const [showOldPassword, setShowOldPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  // District users with default password must change before continuing
  const [requirePasswordChange, setRequirePasswordChange] = useState(false);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setForm((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const { token, user } = await UsersApi.login(form);
      localStorage.setItem("token", token);
      localStorage.setItem("user", JSON.stringify(user));

      // District role with default password: force password change before continuing
      const role = user?.role?.toLowerCase?.() ?? user?.role ?? "";
      if (role === "district" && user.must_change_password) {
        setShowChangePasswordModal(true);
        setRequirePasswordChange(true);
        setChangePasswordForm({
          emailOrUsername: form.emailOrUsername,
          oldPassword: form.password,
          newPassword: "",
        });
        setChangePasswordError("");
        setChangePasswordSuccess("");
        setLoading(false);
        return;
      }

      const route = getRoleRoute(user.role);
      window.location.href = route;
    } catch (e) {
      setError(e.response?.data?.error || e.message || "Login failed");
      setLoading(false);
    }
  };

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };

  const handleChangePasswordClick = (e) => {
    e.preventDefault();
    setRequirePasswordChange(false);
    setShowChangePasswordModal(true);
    setChangePasswordError("");
    setChangePasswordSuccess("");
    setChangePasswordForm({
      emailOrUsername: "",
      oldPassword: "",
      newPassword: "",
    });
  };

  const handleChangePasswordFormChange = (e) => {
    const { name, value } = e.target;
    setChangePasswordForm((prev) => ({ ...prev, [name]: value }));
    setChangePasswordError("");
    setChangePasswordSuccess("");
  };

  const handleChangePasswordSubmit = async (e) => {
    e.preventDefault();
    setChangePasswordError("");
    setChangePasswordSuccess("");
    setChangePasswordLoading(true);

    // Validation
    if (!changePasswordForm.emailOrUsername || !changePasswordForm.oldPassword || !changePasswordForm.newPassword) {
      setChangePasswordError("All fields are required");
      setChangePasswordLoading(false);
      return;
    }

    if (changePasswordForm.newPassword.length < 6) {
      setChangePasswordError("New password must be at least 6 characters long");
      setChangePasswordLoading(false);
      return;
    }

    try {
      const res = await UsersApi.changePassword({
        emailOrUsername: changePasswordForm.emailOrUsername,
        oldPassword: changePasswordForm.oldPassword,
        newPassword: changePasswordForm.newPassword,
      });
      setChangePasswordSuccess("Password changed successfully! You can now login with your new password.");
      if (requirePasswordChange) {
        const updatedUser = res?.user || JSON.parse(localStorage.getItem("user") || "{}");
        if (updatedUser && typeof updatedUser === "object") {
          updatedUser.must_change_password = false;
          localStorage.setItem("user", JSON.stringify(updatedUser));
        }
        setShowChangePasswordModal(false);
        setRequirePasswordChange(false);
        const route = getRoleRoute(updatedUser?.role || "district");
        window.location.href = route;
        return;
      }
      setChangePasswordForm({
        emailOrUsername: "",
        oldPassword: "",
        newPassword: "",
      });
      setTimeout(() => {
        setShowChangePasswordModal(false);
        setChangePasswordSuccess("");
      }, 2000);
    } catch (e) {
      setChangePasswordError(e.response?.data?.error || e.message || "Failed to change password");
    } finally {
      setChangePasswordLoading(false);
    }
  };

  const closeChangePasswordModal = () => {
    if (requirePasswordChange) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
    }
    setRequirePasswordChange(false);
    setShowChangePasswordModal(false);
    setChangePasswordError("");
    setChangePasswordSuccess("");
    setChangePasswordForm({
      emailOrUsername: "",
      oldPassword: "",
      newPassword: "",
    });
  };

  return (
    <div className="main-container">
      <div className="login-container">
        <div className="login-header">
          <img src={require("./logo.png")} alt="StatGate logo" className="brand-logo" />
          <h1>StatGate</h1>
          <p>Field Operations &amp; Agent Workforce Registry</p>
        </div>

        <div className="login-card">
          <h2 className="login-title">Sign In</h2>
          {error && (
            <div className="error-message show">
              <i className="bi bi-exclamation-circle"></i> {error}
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Email or Username</label>
              <div className="input-group">
                <i className="bi bi-person input-icon"></i>
                <input
                  type="text"
                  className="form-control"
                  name="emailOrUsername"
                  value={form.emailOrUsername}
                  onChange={handleChange}
                  placeholder="Enter your email or username"
                  required
                  disabled={loading}
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Password</label>
              <div className="input-group">
                <i className="bi bi-lock input-icon"></i>
                <input
                  type={showPassword ? "text" : "password"}
                  className="form-control"
                  name="password"
                  value={form.password}
                  onChange={handleChange}
                  placeholder="Enter your password"
                  required
                  disabled={loading}
                />
                <button
                  type="button"
                  className="password-toggle"
                  onClick={togglePasswordVisibility}
                  disabled={loading}
                >
                  <i className={showPassword ? "bi bi-eye-slash" : "bi bi-eye"}></i>
                </button>
              </div>
            </div>

            <div className="form-options">
              <label className="remember-me">
                <input
                  type="checkbox"
                  checked={rememberMe}
                  onChange={(e) => setRememberMe(e.target.checked)}
                  disabled={loading}
                />
                <span>Remember me</span>
              </label>
              <button
                type="button"
                onClick={handleChangePasswordClick}
                className="forgot-password"
                style={{ background: "none", border: "none", cursor: "pointer", padding: 0 }}
              >
                Change Password
              </button>
            </div>
            <button
              type="submit"
              className={`btn-login ${loading ? "loading" : ""}`}
              disabled={loading}
            >
              {loading ? "Signing in..." : "Sign In"}
            </button>
          </form>
          <div className="divider">
            <span>OR</span>
          </div>
          <div className="public-access">
            <Link to="/" className="btn-public">
              <i className="bi bi-globe"></i>
              Browse Public Registry
            </Link>
          </div>
        </div>
      </div>

      {/* Change Password Modal (required for district default password, or optional from link) */}
      {showChangePasswordModal && (
        <div className="modal-overlay" onClick={requirePasswordChange ? undefined : closeChangePasswordModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="modal-title">
                {requirePasswordChange ? "Set your password" : "Change Password"}
              </h2>
              {!requirePasswordChange && (
                <button
                  type="button"
                  className="modal-close"
                  onClick={closeChangePasswordModal}
                  aria-label="Close"
                >
                  <i className="bi bi-x"></i>
                </button>
              )}
            </div>
            <div className="modal-body">
              {requirePasswordChange && (
                <p className="password-change-required-msg">
                  You are using the default password. Please set your own password to continue.
                </p>
              )}
              {changePasswordError && (
                <div className="error-message show">
                  <i className="bi bi-exclamation-circle"></i> {changePasswordError}
                </div>
              )}
              {changePasswordSuccess && (
                <div className="success-message show">
                  <i className="bi bi-check-circle"></i> {changePasswordSuccess}
                </div>
              )}
              <form onSubmit={handleChangePasswordSubmit}>
                <div className="form-group">
                  <label className="form-label">Email or Username</label>
                  <div className="input-group">
                    <i className="bi bi-person input-icon"></i>
                    <input
                      type="text"
                      className="form-control"
                      name="emailOrUsername"
                      value={changePasswordForm.emailOrUsername}
                      onChange={handleChangePasswordFormChange}
                      placeholder="Enter your email or username"
                      required
                      disabled={changePasswordLoading}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label className="form-label">Old Password</label>
                  <div className="input-group">
                    <i className="bi bi-lock input-icon"></i>
                    <input
                      type={showOldPassword ? "text" : "password"}
                      className="form-control"
                      name="oldPassword"
                      value={changePasswordForm.oldPassword}
                      onChange={handleChangePasswordFormChange}
                      placeholder="Enter your old password"
                      required
                      disabled={changePasswordLoading}
                    />
                    <button
                      type="button"
                      className="password-toggle"
                      onClick={() => setShowOldPassword(!showOldPassword)}
                      disabled={changePasswordLoading}
                    >
                      <i className={showOldPassword ? "bi bi-eye-slash" : "bi bi-eye"}></i>
                    </button>
                  </div>
                </div>

                <div className="form-group">
                  <label className="form-label">New Password</label>
                  <div className="input-group">
                    <i className="bi bi-lock input-icon"></i>
                    <input
                      type={showNewPassword ? "text" : "password"}
                      className="form-control"
                      name="newPassword"
                      value={changePasswordForm.newPassword}
                      onChange={handleChangePasswordFormChange}
                      placeholder="Enter your new password"
                      required
                      disabled={changePasswordLoading}
                    />
                    <button
                      type="button"
                      className="password-toggle"
                      onClick={() => setShowNewPassword(!showNewPassword)}
                      disabled={changePasswordLoading}
                    >
                      <i className={showNewPassword ? "bi bi-eye-slash" : "bi bi-eye"}></i>
                    </button>
                  </div>
                </div>

                <button
                  type="submit"
                  className={`btn-login ${changePasswordLoading ? "loading" : ""}`}
                  disabled={changePasswordLoading}
                >
                  {changePasswordLoading ? "Changing password..." : "Change Password"}
                </button>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
