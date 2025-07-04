import { useState, useEffect, useCallback } from 'react';

// Global counter for CryptoCompare API requests
let globalCryptoCompareRequestCounter = 0;
const cryptoCompareRequestListeners = new Set();

// Function to increment the counter
const incrementCryptoCompareCounter = () => {
  globalCryptoCompareRequestCounter++;
  // Notify all listeners about the change
  cryptoCompareRequestListeners.forEach(listener => listener(globalCryptoCompareRequestCounter));
};

// Function to get current counter value
export const getCryptoCompareRequestCounter = () => globalCryptoCompareRequestCounter;

// Function to subscribe to counter changes
export const subscribeToCryptoCompareCounter = (listener) => {
  cryptoCompareRequestListeners.add(listener);
  return () => cryptoCompareRequestListeners.delete(listener);
};

const useCryptoCompareHistory = (symbol, timeRange) => {
  const [data, setData] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchCryptoCompareData = useCallback(async () => {
    if (!symbol || !timeRange) return;

    setIsLoading(true);
    setError(null);

    try {
      // Parameters for different time ranges
      const timeRanges = {
        week: { aggregate: 1, limit: 168 }, // 168 hours = 7 days
        month: { aggregate: 4, limit: 180 }, // 4-hour intervals for 30 days  
        halfyear: { aggregate: 1, limit: 180 }, // daily for 6 months
        year: { aggregate: 1, limit: 365 }, // daily for 1 year
        all: { aggregate: 7, limit: 365 } // weekly for max data
      };

      const range = timeRanges[timeRange];
      if (!range) {
        throw new Error('Invalid time range');
      }

      // Increment counter before making the request
      incrementCryptoCompareCounter();

      let endpoint;
      if (timeRange === 'week' || timeRange === 'month') {
        // Use hourly data for short time ranges
        endpoint = `https://min-api.cryptocompare.com/data/v2/histohour?fsym=${symbol.toUpperCase()}&tsym=USD&limit=${range.limit}&aggregate=${range.aggregate}`;
      } else {
        // Use daily data for longer time ranges
        endpoint = `https://min-api.cryptocompare.com/data/v2/histoday?fsym=${symbol.toUpperCase()}&tsym=USD&limit=${range.limit}&aggregate=${range.aggregate}`;
      }

      const response = await fetch(endpoint);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();

      if (result.Response === 'Error') {
        throw new Error(result.Message || 'CryptoCompare API error');
      }

      // Transform data into chart format
      const formattedData = result.Data?.Data?.map((item) => ({
        date: new Date(item.time * 1000).toLocaleDateString(),
        fullDate: new Date(item.time * 1000).toLocaleString(),
        price: item.close,
        timestamp: item.time * 1000,
        high: item.high,
        low: item.low,
        open: item.open,
        volume: item.volumeto
      })) || [];

      setData(formattedData);
    } catch (err) {
      console.error('Error fetching CryptoCompare data:', err);
      setError(err.message || 'Failed to fetch CryptoCompare data');
    } finally {
      setIsLoading(false);
    }
  }, [symbol, timeRange]);

  useEffect(() => {
    fetchCryptoCompareData();
  }, [fetchCryptoCompareData]);

  return { data, isLoading, error, refetch: fetchCryptoCompareData };
};

export default useCryptoCompareHistory; 