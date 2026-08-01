import { apiClient } from "./client";

export const RequestsApi = {
  list: async (params = {}) => {
    const { data } = await apiClient.get("/requests", { params });
    return data;
  },
  getStats: async () => {
    const { data } = await apiClient.get("/requests/stats");
    return data;
  },
  get: async (id) => {
    const { data } = await apiClient.get(`/requests/${id}`);
    return data;
  },
  getDistrictInfo: async () => {
    const { data } = await apiClient.get("/requests/district-info");
    return data;
  },
  getFacilities: async (searchQuery = "") => {
    const { data } = await apiClient.get("/requests/facilities", {
      params: { q: searchQuery },
    });
    return data;
  },
  create: async (requestType, payload, documents = [], facilityMflUid = null) => {
    const formData = new FormData();
    formData.append("request_type", requestType);
    if (facilityMflUid) {
      formData.append("facility_mfl_uid", facilityMflUid);
    }
    if (payload) {
      formData.append("facility_data", JSON.stringify(payload));
    }
    documents.forEach((doc) => {
      // Support both plain File[] and { file, type } objects
      if (doc && doc.file instanceof File) {
        formData.append("documents", doc.file);
        formData.append("document_types", doc.type || "");
      } else {
        formData.append("documents", doc);
      }
    });
    const { data } = await apiClient.post("/requests", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    return data;
  },
  approve: async (id, { comments } = {}) => {
    const { data } = await apiClient.post(`/requests/${id}/approve`, { comments });
    return data;
  },
  reject: async (id, { rejection_reason, comments } = {}) => {
    const { data } = await apiClient.post(`/requests/${id}/reject`, { rejection_reason, comments });
    return data;
  },
  downloadDocument: async (requestId, docId) => {
    const response = await apiClient.get(`/requests/${requestId}/documents/${docId}`, {
      responseType: "blob",
    });
    return response.data;
  },
};
