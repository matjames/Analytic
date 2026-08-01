import { apiClient, publicApiClient } from "./client";

export const FacilitiesApi = {
  list: async (params = {}) => {
    // Use /facilities/ endpoint which uses mfl_details for UI
    const { data } = await apiClient.get("/facilities", { params });
    return data.rows || data;
  },
  listPaged: async (params = {}) => {
    // Use /facilities/ endpoint which uses mfl_details for UI
    const { data } = await apiClient.get("/facilities", { params });
    return data;
  },
  get: async (id, params = {}) => {
    const { data } = await apiClient.get(`/facilities/${id}`, { params });
    return data;
  },
  create: async (payload) => {
    const { data } = await apiClient.post("/facilities", payload);
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/facilities/${id}`, payload);
    return data;
  },
  remove: async (id) => {
    const { data } = await apiClient.delete(`/facilities/${id}`);
    return data;
  },
  export: async (params = {}) => {
    const { data } = await apiClient.get("/facilities/export", { params });
    return data;
  },
  /** Bulk upload facilities from parsed rows. Each row: { name, organisation_unit_id? }. Assigns unique mfl_uid per row. */
  upload: async (facilities) => {
    const { data } = await apiClient.post("/facilities/upload", { facilities });
    return data;
  },
};

// Public, unauthenticated endpoints (no auth interceptor, no 401 redirects)
export const PublicFacilitiesApi = {
  listPaged: async (params = {}) => {
    const { data } = await publicApiClient.get("/facilities/public", { params });
    return data;
  },
  get: async (id, params = {}) => {
    const { data } = await publicApiClient.get(`/facilities/public/${id}`, { params });
    return data;
  },
  export: async (params = {}) => {
    const { data } = await publicApiClient.get("/facilities/public/export", { params });
    return data;
  },
  getDistributionByOwnership: async () => {
    const { data } = await publicApiClient.get("/facilities/distribution/ownership");
    return data;
  },
  getDistributionByLevel: async () => {
    const { data } = await publicApiClient.get("/facilities/distribution/level");
    return data;
  },
  getDistributionByAuthority: async () => {
    const { data } = await publicApiClient.get("/facilities/distribution/authority");
    return data;
  },
};

// StatGate Field Operations Domain Aliases
export const FieldAgentsApi = FacilitiesApi;
export const PublicFieldAgentsApi = PublicFacilitiesApi;
