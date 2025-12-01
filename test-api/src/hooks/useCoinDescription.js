import { useState, useCallback, useRef } from 'react';
import { proxyGet } from '../utils/proxy_request';

/**
 * Hook for fetching coin descriptions on demand
 * Caches results to avoid redundant API calls
 */
export default function useCoinDescription() {
  const [descriptions, setDescriptions] = useState({});
  const [loading, setLoading] = useState({});
  const cache = useRef({});

  const fetchDescription = useCallback(async (coinId) => {
    // Return cached result if available
    if (cache.current[coinId]) {
      return cache.current[coinId];
    }

    // Don't fetch if already loading
    if (loading[coinId]) {
      return null;
    }

    setLoading(prev => ({ ...prev, [coinId]: true }));

    try {
      const response = await proxyGet(`/v1/coins/${coinId}`);
      
      if (response.data) {
        // CoinGecko returns description as { en: "...", de: "...", ... }
        const description = response.data.description?.en || 
                           response.data.description || 
                           'No description available';
        
        // Strip HTML tags from description
        const cleanDescription = typeof description === 'string' 
          ? description.replace(/<[^>]*>/g, '').trim()
          : 'No description available';
        
        // Truncate if too long
        const truncatedDescription = cleanDescription.length > 500 
          ? cleanDescription.substring(0, 500) + '...'
          : cleanDescription;
        
        cache.current[coinId] = truncatedDescription;
        setDescriptions(prev => ({ ...prev, [coinId]: truncatedDescription }));
        
        return truncatedDescription;
      }
    } catch (error) {
      console.error(`Error fetching description for ${coinId}:`, error);
      cache.current[coinId] = 'Failed to load description';
      setDescriptions(prev => ({ ...prev, [coinId]: 'Failed to load description' }));
    } finally {
      setLoading(prev => ({ ...prev, [coinId]: false }));
    }
    
    return null;
  }, [loading]);

  return {
    descriptions,
    loading,
    fetchDescription
  };
}

