import { useEffect } from 'react';
import useApiRequest from './useApiRequest';

export default function useCoinGeckoPriceData(endpoint = 'prices') {
  const endpointUrls = {
    'prices': '/v1/leaderboard/prices',      // by symbol (binance compatible)
    'simpleprices': '/v1/leaderboard/simpleprices'  // by token ID
  };

  const {
    data: coinGeckoPriceData,
    isLoading,
    error,
    stats,
    fetchData,
    resetStats
  } = useApiRequest({
    url: `${process.env.REACT_APP_API_URL}${endpointUrls[endpoint]}`,
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
  }, [endpoint]); // Add endpoint to dependencies

  return { coinGeckoPriceData: coinGeckoPriceData || {}, isLoading, error, stats };
} 