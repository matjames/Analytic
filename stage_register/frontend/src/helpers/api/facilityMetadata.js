import { apiClient, publicApiClient } from "./client";

export const FacilityLevelsApi = {
  list: async () => {
    const { data } = await apiClient.get("/level");
    return data;
  },
  get: async (id) => {
    const { data } = await apiClient.get(`/level/${id}`);
    return data;
  },
  create: async ({ code, name, description }) => {
    const { data } = await apiClient.post("/level", { code, name, description });
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/level/${id}`, payload);
    return data;
  },
  remove: async (id) => {
    const { data } = await apiClient.delete(`/level/${id}`);
    return data;
  },
};

// Public lookup endpoints (no auth)
export const PublicFacilityLevelsApi = {
  list: async () => {
    const { data } = await publicApiClient.get("/level");
    return data;
  },
};

export const PublicOwnershipTypesApi = {
  list: async () => {
    const { data } = await publicApiClient.get("/ownership");
    return data;
  },
};

export const PublicAuthorityTypesApi = {
  list: async () => {
    const { data } = await publicApiClient.get("/authority");
    return data;
  },
};

export const PublicAdminUnitsApi = {
  list: async (params = {}) => {
    const { data } = await publicApiClient.get("/adminunits", { params });
    return data;
  },
  tree: async () => {
    const { data } = await publicApiClient.get("/adminunits/tree/public");
    return data;
  },
};

export const PublicAdminLevelsApi = {
  list: async () => {
    const { data } = await publicApiClient.get("/adminlevel/public");
    return data;
  },
};

export const PublicDistrictsApi = {
  list: async () => {
    const { data } = await publicApiClient.get("/adminunits/districts/public");
    return data;
  },
};

export const OwnershipTypesApi = {
  list: async () => {
    const { data } = await apiClient.get("/ownership");
    return data;
  },
  get: async (id) => {
    const { data } = await apiClient.get(`/ownership/${id}`);
    return data;
  },
  create: async ({ code, name, description }) => {
    const { data } = await apiClient.post("/ownership", { code, name, description });
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/ownership/${id}`, payload);
    return data;
  },
  remove: async (id) => {
    const { data } = await apiClient.delete(`/ownership/${id}`);
    return data;
  },
};

export const AuthorityTypesApi = {
  list: async () => {
    const { data } = await apiClient.get("/authority");
    return data;
  },
  get: async (id) => {
    const { data } = await apiClient.get(`/authority/${id}`);
    return data;
  },
  create: async ({ code, name, description }) => {
    const { data } = await apiClient.post("/authority", { code, name, description });
    return data;
  },
  update: async (id, payload) => {
    const { data } = await apiClient.put(`/authority/${id}`, payload);
    return data;
  },
  remove: async (id) => {
    const { data } = await apiClient.delete(`/authority/${id}`);
    return data;
  },
};

// StatGate Field Operations Domain Aliases
export const FieldRoleTiersApi = FacilityLevelsApi;
export const PublicFieldRoleTiersApi = PublicFacilityLevelsApi;
export const ContractTypesApi = OwnershipTypesApi;
export const PublicContractTypesApi = PublicOwnershipTypesApi;
export const FieldAgenciesApi = AuthorityTypesApi;
export const PublicFieldAgenciesApi = PublicAuthorityTypesApi;
export const FieldTerritoriesApi = PublicAdminUnitsApi;
