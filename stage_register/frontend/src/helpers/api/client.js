import axios from "axios";
import { getValidToken } from "../../utils/auth";

const baseURL =
  process.env.NODE_ENV === "development"
    ? process.env.REACT_APP_API_BASE_URL_DEV || "http://localhost:9090/api"
    : process.env.REACT_APP_API_BASE_URL_PROD || "https://nhfr.health.go.ug/api";

export const apiClient = axios.create({ baseURL });
export const publicApiClient = axios.create({ baseURL });

apiClient.interceptors.request.use(
  (config) => {
    const token = getValidToken();
    if (token) {
      config.headers["Authorization"] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Handle 401 responses - auto logout
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("token");
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
