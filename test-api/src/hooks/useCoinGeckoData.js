import { useEffect, useState, useCallback, useRef } from 'react';
import useApiRequest from './useApiRequest';

export default function useCoinGeckoData(endpoint = 'leaderboard') {
  const [coinIds, setCoinIds] = useState([]);
  const [step, setStep] = useState(endpoint === 'coins' ? 'fetchingIds' : 'ready');
  const intervalRef = useRef(null);
  
  // For leaderboard endpoint - direct request
  const leaderboardRequest = useApiRequest({
    url: '/v1/leaderboard/markets',
    processData: (data) => data.data || [],
    validateData: (data) => {
      return data !== null && 
             typeof data === 'object' && 
             data.data && 
             Array.isArray(data.data);
    },
    silent: false
  });

  // For coins endpoint - request with IDs from leaderboard
  const coinsRequest = useApiRequest({
    url: coinIds.length > 0 ? 
      `/v1/coins/markets?` +
      `ids=${coinIds.join(',')}&` +
      `vs_currency=usd&` +
      `order=market_cap_desc&` +
      `per_page=250&` +
      `page=1&` +
      `sparkline=false&` +
      `price_change_percentage=1h,24h` : null,
    processData: (data) => data || [],
    validateData: (data) => data !== null && Array.isArray(data),
    silent: false
  });

  // Extract coin IDs from leaderboard data
  const extractCoinIds = useCallback((leaderboardData) => {
    if (!Array.isArray(leaderboardData)) return [];
    
    return leaderboardData
      .slice(0, 250) // Get first 250 tokens
      .map(token => token.id)
      .filter(id => id); // Filter out any undefined/null IDs
  }, []);

  // Clear interval on cleanup
  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  // Reset when endpoint changes
  useEffect(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }
    
    leaderboardRequest.resetStats();
    coinsRequest.resetStats();
    setCoinIds([]);
    
    if (endpoint === 'coins') {
      setStep('fetchingIds');
    } else {
      setStep('ready');
    }
  }, [endpoint]);

  // Fetch leaderboard data for coins endpoint to get IDs
  useEffect(() => {
    if (endpoint === 'coins' && step === 'fetchingIds') {
      leaderboardRequest.fetchData();
    }
  }, [endpoint, step]);

  // Extract IDs when leaderboard data is available
  useEffect(() => {
    if (endpoint === 'coins' && step === 'fetchingIds' && leaderboardRequest.data) {
      const ids = extractCoinIds(leaderboardRequest.data);
      setCoinIds(ids);
      setStep('fetchingCoins');
    }
  }, [endpoint, step, leaderboardRequest.data, extractCoinIds]);

  // Fetch coins data when IDs are available
  useEffect(() => {
    if (endpoint === 'coins' && step === 'fetchingCoins' && coinIds.length > 0) {
      coinsRequest.fetchData();
      setStep('ready');
    }
  }, [endpoint, step, coinIds.length]);

  // Initial fetch and interval setup
  useEffect(() => {
    const fetchData = () => {
      if (endpoint === 'leaderboard') {
        leaderboardRequest.fetchData();
      } else if (endpoint === 'coins') {
        if (coinIds.length > 0) {
          coinsRequest.fetchData();
        } else {
          setStep('fetchingIds');
        }
      }
    };

    // Initial fetch
    if (endpoint === 'leaderboard') {
      leaderboardRequest.fetchData();
    }

    // Set up interval
    intervalRef.current = setInterval(fetchData, 30000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [endpoint, coinIds.length]);

  // Return appropriate data based on endpoint
  const currentRequest = endpoint === 'leaderboard' ? leaderboardRequest : coinsRequest;
  const isLoading = endpoint === 'coins' ? 
    (step === 'fetchingIds' ? leaderboardRequest.isLoading : 
     step === 'fetchingCoins' ? coinsRequest.isLoading : 
     coinsRequest.isLoading) :
    leaderboardRequest.isLoading;
  
  return { 
    coinGeckoData: currentRequest.data || [], 
    isLoading,
    error: currentRequest.error,
    stats: currentRequest.stats
  };
} 