import { useState, useEffect, useCallback } from 'react';

// Global counter for local proxy API requests
let globalLocalProxyRequestCounter = 0;
const localProxyRequestListeners = new Set();

// Function to increment the counter
const incrementLocalProxyCounter = () => {
  globalLocalProxyRequestCounter++;
  // Notify all listeners about the change
  localProxyRequestListeners.forEach(listener => listener(globalLocalProxyRequestCounter));
};

// Function to get the current counter value
export const getLocalProxyRequestCounter = () => globalLocalProxyRequestCounter;

// Function to subscribe to counter changes
export const subscribeToLocalProxyCounter = (callback) => {
  localProxyRequestListeners.add(callback);
  return () => localProxyRequestListeners.delete(callback);
};

const useLocalProxyHistory = (tokenId, timeRange) => {
  const [data, setData] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchLocalProxyData = useCallback(async () => {
    if (!tokenId || !timeRange) return;

    setIsLoading(true);
    setError(null);

    try {
      // Parameters for different time ranges
      const timeRanges = {
        week: { days: 7 },
        month: { days: 30 },
        halfyear: { days: 180 },
        year: { days: 365 },
        all: { days: 'max' }
      };

      const range = timeRanges[timeRange];
      if (!range) {
        throw new Error('Invalid time range');
      }

      // Increment counter before making the request
      incrementLocalProxyCounter();

      // Use local proxy API to get market chart data
      const localProxyUrl = process.env.REACT_APP_API_URL || 'http://localhost:8080';
      const response = await fetch(
        `${localProxyUrl}/v1/coins/${tokenId}/market_chart?vs_currency=usd&days=${range.days}`
      );

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();

      // Transform data into chart format
      const formattedData = result.prices?.map(([timestamp, price]) => ({
        date: new Date(timestamp).toLocaleDateString(),
        fullDate: new Date(timestamp).toLocaleString(),
        price: price,
        timestamp: timestamp
      })) || [];

      setData(formattedData);
    } catch (err) {
      console.error('Error fetching local proxy data:', err);
      setError(err.message || 'Failed to fetch local proxy data');
    } finally {
      setIsLoading(false);
    }
  }, [tokenId, timeRange]);

  useEffect(() => {
    fetchLocalProxyData();
  }, [fetchLocalProxyData]);

  return { data, isLoading, error, refetch: fetchLocalProxyData };
};

export default useLocalProxyHistory; 