import { useEffect } from 'react';
import useApiRequest from './useApiRequest';

export default function useCoinGeckoPriceData() {
  const {
    data: coinGeckoPriceData,
    isLoading,
    error,
    stats,
    fetchData
  } = useApiRequest({
    url: `${process.env.REACT_APP_API_URL}/v1/leaderboard/prices`,
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
    fetchData();
    const interval = setInterval(fetchData, 1000); // Fetch every second
    
    return () => clearInterval(interval);
  }, []);

  return { coinGeckoPriceData: coinGeckoPriceData || {}, isLoading, error, stats };
} 