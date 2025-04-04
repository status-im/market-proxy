import { useEffect } from 'react';
import useApiRequest from './useApiRequest';

export default function useCoinGeckoData() {
  const {
    data: coinGeckoData,
    isLoading,
    error,
    stats,
    fetchData
  } = useApiRequest({
    url: `${process.env.REACT_APP_API_URL}/v1/leaderboard/markets`,
    processData: (data) => data.data || [],
    validateData: (data) => {
      // Check response structure {data: Array}
      return data !== null && 
             typeof data === 'object' && 
             data.data && 
             Array.isArray(data.data);
    },
    silent: false // Temporarily enable logs for debugging
  });

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000); // Fetch every 30 seconds
    return () => clearInterval(interval);
  }, []); // Add empty dependencies array

  return { coinGeckoData: coinGeckoData || [], isLoading, error, stats };
} 