import React, { useEffect, useState } from "react";
import { Link, useHistory } from "react-router-dom";
import { UsersApi, PublicDistrictsApi } from "../../helpers/api";
import Header from "../public/Header.jsx";
import "./styles.css";

export default function Register() {
  const history = useHistory();
  const [form, setForm] = useState({
    first_name: "",
    last_name: "",
    username: "",
    password: "",
    confirmPassword: "",
    phoneno: "",
    email: "",
    district_id: "",
    organisation: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [agree, setAgree] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [passwordStrength, setPasswordStrength] = useState({ level: "", text: "" });
  const [fieldErrors, setFieldErrors] = useState({});
  const [districts, setDistricts] = useState([]);
  const [loadingDistricts, setLoadingDistricts] = useState(false);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setForm((prev) => ({ ...prev, [name]: value }));
    
    // Clear field error when user types
    if (fieldErrors[name]) {
      setFieldErrors((prev) => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }

    // Update password strength
    if (name === "password") {
      calculatePasswordStrength(value);
    }
  };

  const calculatePasswordStrength = (password) => {
    if (!password) {
      setPasswordStrength({ level: "", text: "Password strength" });
      return;
    }

    let strength = 0;
    if (password.length >= 8) strength++;
    if (password.length >= 12) strength++;
    if (/[a-z]/.test(password)) strength++;
    if (/[A-Z]/.test(password)) strength++;
    if (/[0-9]/.test(password)) strength++;
    if (/[^a-zA-Z0-9]/.test(password)) strength++;

    if (strength <= 2) {
      setPasswordStrength({ level: "weak", text: "Weak password" });
    } else if (strength <= 4) {
      setPasswordStrength({ level: "medium", text: "Medium password" });
    } else {
      setPasswordStrength({ level: "strong", text: "Strong password" });
    }
  };

  // Load districts for dropdown
  useEffect(() => {
    async function loadDistricts() {
      setLoadingDistricts(true);
      try {
        const districtsList = await PublicDistrictsApi.list();
        // The registry API exposes StatGate terminology (territory_name and
        // statgate_uid); accept the legacy aliases too for backward compatibility.
        const normalizedDistricts = (districtsList || [])
          .map((district) => ({
            ...district,
            name: district.territory_name || district.name || "",
            mfl_uid: district.statgate_uid || district.mfl_uid || String(district.id || ""),
          }))
          .filter((district) => district.name && district.mfl_uid);
        setDistricts(normalizedDistricts);
      } catch (e) {
        console.error("Failed to load districts for register:", e);
        setError("Failed to load districts. Please refresh the page.");
      } finally {
        setLoadingDistricts(false);
      }
    }
    loadDistricts();
  }, []);

  const validateForm = () => {
    const errors = {};

    if (!form.email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
      errors.email = "Please enter a valid email address";
    }

    if (!form.first_name || form.first_name.trim().length < 2) {
      errors.first_name = "First name must be at least 2 characters";
    }

    if (!form.last_name || form.last_name.trim().length < 2) {
      errors.last_name = "Last name must be at least 2 characters";
    }

    if (!form.username || form.username.length < 4) {
      errors.username = "Username must be at least 4 characters";
    }

    if (!form.organisation || form.organisation.trim() === "") {
      errors.organisation = "Organization is required";
    }

    if (!form.phoneno || form.phoneno.trim() === "") {
      errors.phoneno = "Phone number is required";
    }

    if (!form.district_id || form.district_id.trim() === "") {
      errors.district_id = "District is required";
    }

    if (!form.password || form.password.length < 6) {
      errors.password = "Password must be at least 6 characters";
    }

    if (form.password !== form.confirmPassword) {
      errors.confirmPassword = "Passwords do not match";
    }

    if (!agree) {
      setError("Please accept the terms to continue.");
      return false;
    }

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setSuccess("");

    if (!validateForm()) {
      return;
    }

    setLoading(true);
    try {
      await UsersApi.register({
        first_name: form.first_name || null,
        last_name: form.last_name || null,
        username: form.username,
        password: form.password,
        email: form.email,
        phoneno: form.phoneno || null,
        district_id: form.district_id || null,
        organisation: form.organisation || null,
        role: "public",
      });
      setSuccess("Registration successful! You will be redirected to the login page.");
      // Redirect to login page after showing success message
      setTimeout(() => {
        history.push("/login");
      }, 1500);
    } catch (e) {
      setError(e.response?.data?.error || e.message || "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  const togglePasswordVisibility = () => {
    setShowPassword(!showPassword);
  };

  const toggleConfirmPasswordVisibility = () => {
    setShowConfirmPassword(!showConfirmPassword);
  };

  return (
    <>
      <Header />
      <div className="register-container">
        <div className="register-card">
          <div className="register-header">
            <img src="/statgate-logo.svg" alt="StatGate logo" className="brand-logo" />
            <h1>StatGate</h1>
            <p>Create your account</p>
          </div>
          <h2 className="register-title">Create Account</h2>

          {success && (
            <div className="alert alert-success alert-dismissible fade show" role="alert">
              <i className="bi bi-check-circle"></i> {success}
            </div>
          )}

          {error && (
            <div className="alert alert-danger alert-dismissible fade show" role="alert">
              <i className="bi bi-exclamation-circle"></i> {error}
              <button
                type="button"
                className="btn-close"
                onClick={() => setError("")}
                aria-label="Close"
              />
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Email Address <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-envelope input-icon"></i>
                  <input
                    type="email"
                    name="email"
                    value={form.email}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.email ? "error" : ""}`}
                    placeholder="your.email@example.com"
                    required
                  />
                </div>
                {fieldErrors.email && (
                  <div className="field-error show">{fieldErrors.email}</div>
                )}
              </div>

              <div className="form-group">
                <label className="form-label">First Name <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-person input-icon"></i>
                  <input
                    type="text"
                    name="first_name"
                    value={form.first_name}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.first_name ? "error" : ""}`}
                    placeholder="Your first name"
                    required
                  />
                </div>
                {fieldErrors.first_name && (
                  <div className="field-error show">{fieldErrors.first_name}</div>
                )}
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Last Name <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-person input-icon"></i>
                  <input
                    type="text"
                    name="last_name"
                    value={form.last_name}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.last_name ? "error" : ""}`}
                    placeholder="Your last name"
                    required
                  />
                </div>
                {fieldErrors.last_name && (
                  <div className="field-error show">{fieldErrors.last_name}</div>
                )}
              </div>

              <div className="form-group">
                <label className="form-label">Username <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-person input-icon"></i>
                  <input
                    type="text"
                    name="username"
                    value={form.username}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.username ? "error" : ""}`}
                    placeholder="Choose a username"
                    required
                  />
                </div>
                {fieldErrors.username && (
                  <div className="field-error show">{fieldErrors.username}</div>
                )}
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Organization <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-building input-icon"></i>
                  <input
                    type="text"
                    name="organisation"
                    value={form.organisation}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.organisation ? "error" : ""}`}
                    placeholder="Your organization"
                    required
                  />
                </div>
                {fieldErrors.organisation && (
                  <div className="field-error show">{fieldErrors.organisation}</div>
                )}
              </div>

              <div className="form-group">
                <label className="form-label">Phone Number <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-phone input-icon"></i>
                  <input
                    type="tel"
                    name="phoneno"
                    value={form.phoneno}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.phoneno ? "error" : ""}`}
                    placeholder="e.g. +256700000000"
                    required
                  />
                </div>
                {fieldErrors.phoneno && (
                  <div className="field-error show">{fieldErrors.phoneno}</div>
                )}
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">Password <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-lock input-icon"></i>
                  <input
                    type={showPassword ? "text" : "password"}
                    name="password"
                    value={form.password}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.password ? "error" : ""}`}
                    placeholder="Create a strong password"
                    required
                  />
                  <button
                    type="button"
                    className="password-toggle"
                    onClick={togglePasswordVisibility}
                  >
                    <i className={`bi ${showPassword ? "bi-eye-slash" : "bi-eye"}`}></i>
                  </button>
                </div>
                {form.password && (
                  <div className="password-strength">
                    <div className={`strength-bar ${passwordStrength.level}`}>
                      <span></span>
                      <span></span>
                      <span></span>
                    </div>
                    <div className={`strength-text ${passwordStrength.level}`}>
                      {passwordStrength.text}
                    </div>
                  </div>
                )}
                {fieldErrors.password && (
                  <div className="field-error show">{fieldErrors.password}</div>
                )}
              </div>

              <div className="form-group">
                <label className="form-label">Confirm Password <span className="required">*</span></label>
                <div className="input-group">
                  <i className="bi bi-lock input-icon"></i>
                  <input
                    type={showConfirmPassword ? "text" : "password"}
                    name="confirmPassword"
                    value={form.confirmPassword}
                    onChange={handleChange}
                    className={`form-control ${fieldErrors.confirmPassword ? "error" : ""}`}
                    placeholder="Re-enter your password"
                    required
                  />
                  <button
                    type="button"
                    className="password-toggle"
                    onClick={toggleConfirmPasswordVisibility}
                  >
                    <i className={`bi ${showConfirmPassword ? "bi-eye-slash" : "bi-eye"}`}></i>
                  </button>
                </div>
                {fieldErrors.confirmPassword && (
                  <div className="field-error show">{fieldErrors.confirmPassword}</div>
                )}
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">District <span className="required">*</span></label>
              <div className="input-group">
                <i className="bi bi-geo-alt input-icon"></i>
                <select
                  name="district_id"
                  value={form.district_id}
                  onChange={handleChange}
                  className={`form-control ${fieldErrors.district_id ? "error" : ""}`}
                  required
                  disabled={loadingDistricts}
                >
                  <option value="">{loadingDistricts ? "Loading districts..." : "Select your district"}</option>
                  {districts.map((district) => (
                    <option key={district.mfl_uid || district.id} value={district.mfl_uid}>
                      {district.name}
                    </option>
                  ))}
                </select>
              </div>
              {fieldErrors.district_id && (
                <div className="field-error show">{fieldErrors.district_id}</div>
              )}
            </div>

            <label className="terms-checkbox">
              <input
                type="checkbox"
                checked={agree}
                onChange={(e) => setAgree(e.target.checked)}
                required
              />
              <span>I agree to the <a href="/terms" onClick={(e) => e.preventDefault()}>Terms of Service</a> and <a href="/privacy" onClick={(e) => e.preventDefault()}>Privacy Policy</a></span>
            </label>

            <button
              type="submit"
              className={`btn-register ${loading ? "loading" : ""}`}
              disabled={loading}
            >
              {loading ? "Creating Account..." : "Create Account"}
            </button>
          </form>

          <div className="divider">
            <span>Already have an account?</span>
          </div>

          <div className="login-link">
            <Link to="/login">Sign in here</Link>
          </div>
        </div>
      </div>
    </>
  );
}
