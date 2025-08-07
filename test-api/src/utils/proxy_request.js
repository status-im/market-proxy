import axios from 'axios';

/**
 * Utility function for making authenticated requests to the proxy server
 * @param {string} endpoint - The API endpoint path (e.g., '/v1/prices')
 * @param {Object} options - Request options
 * @param {Object} options.headers - Additional headers to include
 * @param {Object} options.params - URL parameters
 * @param {string} options.method - HTTP method (default: 'GET')
 * @param {Object} options.data - Request body data for POST/PUT requests
 * @param {function} options.validateStatus - Custom status validation function
 * @param {boolean} options.useAxios - Whether to use axios (default: true) or fetch
 * @returns {Promise} - Promise resolving to the response
 */
export const makeProxyRequest = async (endpoint, options = {}) => {
  const {
    headers = {},
    params = {},
    method = 'GET',
    data = null,
    validateStatus = (status) => (status >= 200 && status < 300) || status === 304,
    useAxios = true
  } = options;

  // Construct full URL
  const baseUrl = process.env.REACT_APP_API_URL || 'http://localhost:8080';
  const fullUrl = `${baseUrl}${endpoint}`;

  // Prepare authentication
  const auth = {
    username: process.env.REACT_APP_PROXY_USER,
    password: process.env.REACT_APP_PROXY_PASSWORD
  };

  if (useAxios) {
    // Use axios for requests (preferred for useApiRequest)
    const config = {
      method,
      url: fullUrl,
      headers,
      params,
      auth,
      validateStatus
    };

    if (data && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
      config.data = data;
    }

    return await axios(config);
  } else {
    // Use fetch for requests (for RequestReplay compatibility)
    const fetchHeaders = {
      ...headers
    };

    // Add basic auth header for fetch
    if (auth.username && auth.password) {
      const credentials = btoa(`${auth.username}:${auth.password}`);
      fetchHeaders['Authorization'] = `Basic ${credentials}`;
    }

    // Construct URL with params
    const url = new URL(fullUrl);
    Object.entries(params).forEach(([key, value]) => {
      if (value !== null && value !== undefined) {
        url.searchParams.append(key, value);
      }
    });

    const fetchConfig = {
      method,
      headers: fetchHeaders
    };

    if (data && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
      fetchConfig.body = JSON.stringify(data);
      fetchHeaders['Content-Type'] = 'application/json';
    }

    const response = await fetch(url.toString(), fetchConfig);
    
    // Apply status validation similar to axios
    if (!validateStatus(response.status)) {
      throw new Error(`Request failed with status ${response.status}`);
    }

    return response;
  }
};

/**
 * Convenience function for making GET requests with axios (for useApiRequest)
 * @param {string} endpoint - The API endpoint path
 * @param {Object} options - Request options (headers, params, etc.)
 * @returns {Promise} - Promise resolving to axios response
 */
export const proxyGet = (endpoint, options = {}) => {
  return makeProxyRequest(endpoint, { ...options, method: 'GET', useAxios: true });
};

/**
 * Convenience function for making GET requests with fetch (for RequestReplay)
 * @param {string} endpoint - The API endpoint path
 * @param {Object} options - Request options (headers, params, etc.)
 * @returns {Promise} - Promise resolving to fetch response
 */
export const proxyFetch = (endpoint, options = {}) => {
  return makeProxyRequest(endpoint, { ...options, method: 'GET', useAxios: false });
};

/**
 * Helper function to extract endpoint from full URL (for RequestReplay)
 * @param {string} fullUrl - The complete URL
 * @returns {string} - The endpoint path
 */
export const extractEndpointFromUrl = (fullUrl) => {
  try {
    const url = new URL(fullUrl);
    return url.pathname + url.search;
  } catch (error) {
    // If URL parsing fails, try to extract endpoint manually
    const apiMatch = fullUrl.match(/\/api\/v1(.*)/) || fullUrl.match(/\/v1(.*)/);
    return apiMatch ? `/v1${apiMatch[1]}` : fullUrl;
  }
};