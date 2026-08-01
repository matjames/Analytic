import { apiClient, publicApiClient } from "./client";

export const UsersApi = {
  register: async (payload) => {
    const { data } = await publicApiClient.post("/users/register", payload);
    return data;
  },
  login: async ({ emailOrUsername, password }) => {
    const { data } = await apiClient.post("/users/login", { emailOrUsername, password });
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
};
