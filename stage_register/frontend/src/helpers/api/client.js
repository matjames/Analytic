import axios from "axios";
import { getValidToken } from "../../utils/auth";

const baseURL =
  process.env.NODE_ENV === "development"
    ? process.env.REACT_APP_API_BASE_URL_DEV || "http://localhost:9090/api"
    : process.env.REACT_APP_API_BASE_URL_PROD || "https://nhfr.health.go.ug/api";

export const apiClient = axios.create({ baseURL });
export const publicApiClient = axios.create({ baseURL });
let refreshPromise = null;

apiClient.interceptors.request.use(
  (config) => {
    const token = getValidToken();
    if (token) {
      config.headers["Authorization"] = `Bearer ${token}`;
    }
    const workspaceID = localStorage.getItem("workspace_id");
    if (workspaceID) config.headers["X-Workspace-ID"] = workspaceID;
    return config;
  },
  (error) => Promise.reject(error)
);

// Handle 401 responses - auto logout
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    const refreshToken = localStorage.getItem("refresh_token");
    const isAuthEndpoint = originalRequest?.url?.includes("/users/login") ||
      originalRequest?.url?.includes("/users/refresh");
    let refreshFailed = false;

    if (error.response?.status === 401 && refreshToken && !originalRequest?._retry && !isAuthEndpoint) {
      originalRequest._retry = true;
      refreshPromise = refreshPromise || publicApiClient
        .post("/users/refresh", { refresh_token: refreshToken })
        .then(({ data }) => {
          localStorage.setItem("token", data.token);
          if (data.refresh_token) localStorage.setItem("refresh_token", data.refresh_token);
          return data.token;
        })
        .finally(() => {
          refreshPromise = null;
        });

      try {
        const token = await refreshPromise;
        originalRequest.headers = originalRequest.headers || {};
        originalRequest.headers.Authorization = `Bearer ${token}`;
        return apiClient(originalRequest);
      } catch (refreshError) {
        // Fall through to the normal logout path when refresh is rejected.
        refreshFailed = true;
        error = refreshError;
      }
    }

    if (error.response?.status === 401 || error.config?._retry || refreshFailed) {
      localStorage.removeItem("token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("user");
      window.dispatchEvent(new CustomEvent("auth-expired"));
      // Only redirect if not already on login page
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  }
);
