import { apiClient } from "./client";

export const LevelsApi = {
  list: async () => {
    const { data } = await apiClient.get("/adminlevel");
    return data;
  },
  create: async ({ name, code }) => {
    const { data } = await apiClient.post("/adminlevel", { name, code });
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/adminlevel/${id}`, payload);
    return data;
  },
  remove: async (id) => {
    const { data } = await apiClient.delete(`/adminlevel/${id}`);
    return data;
  },
  reorder: async (ids) => {
    const { data } = await apiClient.post("/adminlevel/reorder", { ids });
    return data;
  },
  seedDefaults: async () => {
    const { data } = await apiClient.post("/adminlevel/seed");
    return data;
  },
};

export const UnitsApi = {
  list: async (params = {}) => {
    const { data } = await apiClient.get("/adminunits", { params });
    return data;
  },
  listPaged: async (params = {}) => {
    const { data } = await apiClient.get("/adminunits/paged", { params });
    return data;
  },
  tree: async () => {
    const { data } = await apiClient.get("/adminunits/tree");
    return data;
  },
  create: async ({ name, code, levelId, parentId }) => {
    const { data } = await apiClient.post("/adminunits", {
      name,
      code,
      levelId,
      parentId: parentId ?? null, // backend expects parent's numeric id or null
    });
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/adminunits/${id}`, payload);
    return data;
  },
  move: async (id, newParentId) => {
    const { data } = await apiClient.post(`/adminunits/${id}/move`, {
      newParentId: newParentId ?? null, // backend expects parent's numeric id or null
    });
    return data;
  },
  remove: async (id, { cascade = false } = {}) => {
    const { data } = await apiClient.delete(`/adminunits/${id}`, {
      params: cascade ? { cascade: "true" } : {},
    });
    return data;
  },
  uploadCsv: async (formData) => {
    const { data } = await apiClient.post("/adminunits/upload", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    return data;
  },
};

export const DashboardApi = {
  getStats: async () => {
    const { data } = await apiClient.get("/dashboard/stats");
    return data;
  },
};