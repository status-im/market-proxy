import { useState, useRef } from 'react';
import axios from 'axios';

/**
 * A utility hook for making API requests with ETag support and statistics tracking
 * @param {Object} options - Configuration options for the API request
 * @param {string} options.url - The API endpoint URL
 * @param {function} options.processData - Function to process successful responses
 * @param {function} options.validateData - Function to validate the response data
 * @param {Object} options.requestConfig - Additional axios request configuration
 * @param {boolean} options.silent - Flag to disable console logging
 * @returns {Object} - State and fetch function for the API request
 */
export default function useApiRequest({
  url,
  processData,
  validateData,
  requestConfig = {},
  silent = false
}) {
  const [data, setData] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState(null);
  const etagRef = useRef(null);
  
  const initialStats = {
    total_requests: 0,
    cache_hits: 0,
    cache_misses: 0,
    total_response_size: 0,
    not_modified_count: 0
  };
  
  const [stats, setStats] = useState(initialStats);

  // Helper function for conditional logging
  const log = (message, data) => {
    if (!silent) {
      console.log(message, data);
    }
  };

  // Function to reset statistics
  const resetStats = () => {
    setStats(initialStats);
    etagRef.current = null; // Also reset ETag cache
    log('Stats reset', null);
  };

  // Add this function to output headers in raw format
  const logRawHeaders = (headers) => {
    if (silent) return;
    
    console.log('--- RAW HEADERS ---');
    Object.entries(headers).forEach(([key, value]) => {
      console.log(`${key}: ${value}`);
    });
    console.log('------------------');
  };

  const fetchData = async () => {
    try {
      setIsLoading(true);
      setError(null);
      
      // Prepare headers with ETag and gzip support
      const headers = { 
        ...(requestConfig.headers || {})
      };
      if (etagRef.current) {
        headers['If-None-Match'] = etagRef.current;
      }
      log("send etag:", etagRef.current)
      
      // Make the API request
      const response = await axios.get(url, {
        ...requestConfig,
        headers,
        auth: requestConfig.auth || {
          username: process.env.REACT_APP_PROXY_USER,
          password: process.env.REACT_APP_PROXY_PASSWORD
        },
        validateStatus: status => (status >= 200 && status < 300) || status === 304
      });
      
      // Output headers in raw format
      logRawHeaders(response.headers);
      
      // Store the new ETag if provided
      if (response.headers.etag) {
        etagRef.current = response.headers.etag;
      }
      
      // Debug logging for headers
      // log('All response headers:', response.headers);
      // log('X-Proxy-Cache header:', response.headers['x-proxy-cache']);
      // log('Content-Length header:', response.headers['content-length']);
      // log('Content-Encoding header:', response.headers['content-encoding']);
      // log('ETag header:', response.headers.etag);
      // log('Status:', response.status);
      
      // Update stats based on response headers
      setStats(prevStats => {
        const isCacheHit = response.headers['x-proxy-cache'] === 'HIT';
        // Calculate actual response size from the data instead of headers
        let responseSize = 0;
        if (response.status !== 304 && response.data) {
          try {
            responseSize = new Blob([JSON.stringify(response.data)]).size;
          } catch (e) {
            // Fallback to string length if Blob is not available
            responseSize = JSON.stringify(response.data).length;
          }
        }
        const isNotModified = response.status === 304;
        const isCompressed = response.headers['content-encoding'] === 'gzip';
        const originalSize = parseInt(response.headers['x-response-size'] || '0');
        const bytesSaved = isCompressed ? originalSize - responseSize : 0;
        
        // log('Cache hit status:', isCacheHit);
        // log('Response size:', responseSize);
        // log('Not Modified status:', isNotModified);
        // log('Compressed status:', isCompressed);
        // log('Original size:', originalSize);
        // log('Bytes saved:', bytesSaved);
        
        return {
          total_requests: prevStats.total_requests + 1,
          cache_hits: prevStats.cache_hits + (isCacheHit ? 1 : 0),
          cache_misses: prevStats.cache_misses + (isCacheHit ? 0 : 1),
          total_response_size: prevStats.total_response_size + responseSize,
          not_modified_count: prevStats.not_modified_count + (isNotModified ? 1 : 0)
        };
      });
      
      // Only update data if not 304 Not Modified
      if (response.status !== 304) {
        if (validateData(response.data)) {
          const processedData = processData(response.data);
          setData(processedData);
        } else {
          if (!silent) {
            console.error('Invalid data format:', response.data);
          }
          throw new Error('Invalid data format received from API');
        }
      } else {
        log('Using cached data (304 Not Modified)', null);
      }
    } catch (error) {
      if (!silent) {
        console.error('Error fetching data:', error);
      }
      setError(error.message || 'Failed to fetch data. Please try again later.');
    } finally {
      setIsLoading(false);
    }
  };

  return { data, isLoading, error, stats, fetchData, resetStats };
} 