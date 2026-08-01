import { apiClient, publicApiClient } from "./client";

export const DocumentsApi = {
  list: async (category = null) => {
    const params = category ? { category } : {};
    const { data } = await publicApiClient.get("/documents", { params });
    return data;
  },
  get: async (id) => {
    const { data } = await publicApiClient.get(`/documents/${id}`);
    return data;
  },
  upload: async (formData) => {
    const { data } = await apiClient.post("/documents", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    return data;
  },
  update: async (id, updates) => {
    const { data } = await apiClient.put(`/documents/${id}`, updates);
    return data;
  },
  delete: async (id) => {
    const { data } = await apiClient.delete(`/documents/${id}`);
    return data;
  },
  download: async (id) => {
    const response = await publicApiClient.get(`/documents/${id}/download`, {
      responseType: "blob",
    });
    return response.data;
  },
};
