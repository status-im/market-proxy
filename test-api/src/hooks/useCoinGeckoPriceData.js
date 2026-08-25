import { useEffect } from 'react';
import useApiRequest from './useApiRequest';
import { BASE_CURRENCY, convertParam } from '../utils/currencies';

export default function useCoinGeckoPriceData(endpoint = 'prices', currency = BASE_CURRENCY) {
  const query = convertParam(currency);
  const suffix = query ? `?${query}` : '';
  const endpointUrls = {
    'prices': `/v1/leaderboard/prices${suffix}`,      // by symbol (binance compatible)
    'simpleprices': `/v1/leaderboard/simpleprices${suffix}`  // by token ID
  };

  const {
    data: coinGeckoPriceData,
    isLoading,
    error,
    stats,
    fetchData,
    resetStats
  } = useApiRequest({
    url: endpointUrls[endpoint],
    processData: (data) => data || {},
    validateData: (data) => {
      // Check that data exists and is an object with keys
      return data !== null && 
             typeof data === 'object' && 
             !Array.isArray(data) &&
             Object.keys(data).length > 0;
    },
    silent: false // Temporarily enable logs for debugging
  });

  useEffect(() => {
    // Reset stats when endpoint changes
    resetStats();
    fetchData();
    const interval = setInterval(fetchData, 1000); // Fetch every second

    return () => clearInterval(interval);
  }, [endpoint, currency]); // Refetch when the endpoint or the currency changes

  return { coinGeckoPriceData: coinGeckoPriceData || {}, isLoading, error, stats };
} 