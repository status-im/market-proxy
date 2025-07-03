import { useState, useEffect, useCallback } from 'react';

// Global counter for CoinGecko API requests
let globalApiRequestCounter = 0;
const apiRequestListeners = new Set();

// Function to increment the counter
const incrementApiCounter = () => {
  globalApiRequestCounter++;
  // Notify all listeners about the change
  apiRequestListeners.forEach(listener => listener(globalApiRequestCounter));
};

// Function to get current counter value
export const getApiRequestCounter = () => globalApiRequestCounter;

// Function to subscribe to counter changes
export const subscribeToApiCounter = (listener) => {
  apiRequestListeners.add(listener);
  return () => apiRequestListeners.delete(listener);
};

const useTokenHistory = (tokenId, timeRange) => {
  const [data, setData] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchHistoricalData = useCallback(async () => {
    if (!tokenId || !timeRange) return;

    setIsLoading(true);
    setError(null);

    try {
      // Parameters for different time ranges
      const timeRanges = {
        week: { days: 7, interval: 'hourly' },
        month: { days: 30, interval: 'hourly' },
        halfyear: { days: 180, interval: 'daily' },
        year: { days: 365, interval: 'daily' },
        all: { days: 'max', interval: 'daily' }
      };

      const range = timeRanges[timeRange];
      if (!range) {
        throw new Error('Invalid time range');
      }

      // Increment counter before making the request
      incrementApiCounter();

      // Use CoinGecko API to get historical data
      const response = await fetch(
        `https://api.coingecko.com/api/v3/coins/${tokenId}/market_chart?vs_currency=usd&days=${range.days}&interval=${range.interval}`
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
      console.error('Error fetching historical data:', err);
      setError(err.message || 'Failed to fetch historical data');
    } finally {
      setIsLoading(false);
    }
  }, [tokenId, timeRange]);

  useEffect(() => {
    fetchHistoricalData();
  }, [fetchHistoricalData]);

  return { data, isLoading, error, refetch: fetchHistoricalData };
};

export default useTokenHistory; 