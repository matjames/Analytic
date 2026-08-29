import { apiClient, publicApiClient } from "./client";

export const UsersApi = {
  register: async (payload) => {
    const { data } = await publicApiClient.post("/users/register", payload);
    return data;
  },
  verifyEmail: async (token) => {
    const { data } = await publicApiClient.get(`/users/verify-email?token=${encodeURIComponent(token)}`);
    return data;
  },
  resendVerification: async (email) => {
    const { data } = await publicApiClient.post("/users/resend-verification", { email });
    return data;
  },
  login: async ({ emailOrUsername, password, mfaCode }) => {
    const { data } = await apiClient.post("/users/login", {
      emailOrUsername,
      password,
      ...(mfaCode ? { mfa_code: mfaCode } : {}),
    });
    return data;
  },
  refresh: async (refreshToken) => {
    const { data } = await publicApiClient.post("/users/refresh", {
      refresh_token: refreshToken,
    });
    return data;
  },
  logout: async () => {
    const { data } = await apiClient.post("/users/me/logout");
    return data;
  },
  sessions: async () => {
    const { data } = await apiClient.get("/users/me/sessions");
    return data.sessions || [];
  },
  revokeSession: async (id) => {
    const { data } = await apiClient.delete(`/users/me/sessions/${id}`);
    return data;
  },
  preferences: async () => {
    const { data } = await apiClient.get("/users/me/preferences");
    return data.preferences;
  },
  updatePreferences: async (payload) => {
    const { data } = await apiClient.put("/users/me/preferences", payload);
    return data.preferences;
  },
  getBranding: async () => {
    const { data } = await apiClient.get("/organisation/branding");
    return data;
  },
  updateBranding: async (payload) => {
    const { data } = await apiClient.put("/organisation/branding", payload);
    return data;
  },
  mfaSetup: async () => {
    const { data } = await apiClient.post("/users/me/mfa/setup");
    return data;
  },
  mfaEnable: async (code) => {
    const { data } = await apiClient.post("/users/me/mfa/enable", { code });
    return data;
  },
  mfaDisable: async (code) => {
    const { data } = await apiClient.post("/users/me/mfa/disable", { code });
    return data;
  },
  me: async () => {
    const { data } = await apiClient.get("/users/me");
    return data.user;
  },
  list: async () => {
    const { data } = await apiClient.get("/users");
    return data;
  },
  get: async (id) => {
    const { data } = await apiClient.get(`/users/${id}`);
    return data;
  },
  create: async (payload) => {
    // Protected endpoint for creating users (requires authentication)
    const { data } = await apiClient.post("/users", payload);
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/users/${id}`, payload);
    return data;
  },
  remove: async (id) => {
    const { data } = await apiClient.delete(`/users/${id}`);
    return data;
  },
  resetPassword: async (id, newPassword) => {
    const { data } = await apiClient.post(`/users/${id}/reset-password`, { newPassword });
    return data;
  },
  changePassword: async ({ emailOrUsername, oldPassword, newPassword }) => {
    const { data } = await publicApiClient.post("/users/change-password", {
      emailOrUsername,
      oldPassword,
      newPassword,
    });
    return data;
  },
  uploadCsv: async (formData) => {
    const { data } = await apiClient.post("/users/upload", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    return data;
  },
  listInvitations: async () => {
    const { data } = await apiClient.get("/users/invitations");
    return data.invitations || [];
  },
  createInvitation: async (payload) => {
    const { data } = await apiClient.post("/users/invitations", payload);
    return data;
  },
  revokeInvitation: async (id) => {
    const { data } = await apiClient.delete(`/users/invitations/${id}`);
    return data;
  },
  acceptInvitation: async (payload) => {
    const { data } = await publicApiClient.post("/users/invitations/accept", payload);
    return data;
  },
};
