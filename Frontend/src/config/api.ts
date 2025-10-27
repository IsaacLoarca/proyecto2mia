// Configuración de la API
const API_BASE_URL = import.meta.env.VITE_API_URL;

export const API_ENDPOINTS = {
  health: `${API_BASE_URL}/health`,
  login: `${API_BASE_URL}/login`,
  logout: `${API_BASE_URL}/logout`,
  session: `${API_BASE_URL}/session`,
  analizar: `${API_BASE_URL}/analizar`,
  directoryTree: `${API_BASE_URL}/directory-tree`,
} as const;

export { API_BASE_URL };