import { apiClient } from '../api';

const sessionListeners = new Set();
let sessionVersion = 0;

export function observeMasterSession(listener) {
  sessionListeners.add(listener);
  return () => sessionListeners.delete(listener);
}

export function advanceMasterSession() {
  sessionVersion += 1;
}

function request(method) {
  return async (...args) => {
    const version = sessionVersion;
    try {
      return await apiClient[method](...args);
    } catch (error) {
      if (error.response?.status === 401 && version === sessionVersion) {
        advanceMasterSession();
        sessionListeners.forEach((listener) => listener());
      }
      throw error;
    }
  };
}

export const masterApi = {
  get: request('get'),
  post: request('post'),
  put: request('put'),
  patch: request('patch'),
  delete: request('delete'),
};
