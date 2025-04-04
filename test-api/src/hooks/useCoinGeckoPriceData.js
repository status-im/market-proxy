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
      // Проверяем, что data существует и является объектом с ключами
      return data !== null && 
             typeof data === 'object' && 
             !Array.isArray(data) &&
             Object.keys(data).length > 0;
    },
    silent: false // Временно включаем логи для отладки
  });

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 1000); // Fetch every second
    
    return () => clearInterval(interval);
  }, []);

  return { coinGeckoPriceData: coinGeckoPriceData || {}, isLoading, error, stats };
} 