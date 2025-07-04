import React, { useState, useEffect } from 'react';
import styled from 'styled-components';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import useTokenHistory, { getApiRequestCounter, subscribeToApiCounter } from '../hooks/useTokenHistory';
import useCryptoCompareHistory, { getCryptoCompareRequestCounter, subscribeToCryptoCompareCounter } from '../hooks/useCryptoCompareHistory';
import useLocalProxyHistory, { getLocalProxyRequestCounter, subscribeToLocalProxyCounter } from '../hooks/useLocalProxyHistory';

const Container = styled.div`
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
`;

const Modal = styled.div`
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 1200px;
  max-height: 90vh;
  overflow-y: auto;
  padding: 24px;
  position: relative;
`;

const CloseButton = styled.button`
  position: absolute;
  top: 16px;
  right: 16px;
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #666;
  
  &:hover {
    color: #333;
  }
`;

const TokenHeader = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
`;

const TokenImage = styled.img`
  width: 64px;
  height: 64px;
  border-radius: 50%;
`;

const TokenInfo = styled.div`
  flex: 1;
`;

const TokenName = styled.h2`
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
`;

const TokenSymbol = styled.span`
  color: #666;
  font-size: 16px;
  text-transform: uppercase;
`;

const ApiCounter = styled.div`
  margin-left: auto;
  background: #f0f8ff;
  border: 1px solid #3861FB;
  border-radius: 6px;
  padding: 8px 12px;
  font-size: 14px;
  color: #3861FB;
  font-weight: 500;
`;

const PriceInfo = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
`;

const PriceCard = styled.div`
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
`;

const PriceLabel = styled.div`
  color: #666;
  font-size: 14px;
  margin-bottom: 8px;
`;

const PriceValue = styled.div`
  font-size: 20px;
  font-weight: 600;
  color: ${props => props.$isPositive ? '#16C784' : props.$isNegative ? '#EA3943' : '#333'};
`;

const ChartsContainer = styled.div`
  margin-bottom: 24px;
`;

const ChartsGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 24px;
  margin-top: 16px;
  
  @media (max-width: 1200px) {
    grid-template-columns: 1fr 1fr;
  }
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
`;

const ChartContainer = styled.div`
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
`;

const ChartTitle = styled.h3`
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
`;

const TimeRangeSelector = styled.div`
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
`;

const TimeRangeButton = styled.button`
  padding: 8px 16px;
  border: 1px solid #ddd;
  background: ${props => props.$active ? '#3861FB' : 'white'};
  color: ${props => props.$active ? 'white' : '#333'};
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  
  &:hover {
    background: ${props => props.$active ? '#3861FB' : '#f5f5f5'};
  }
`;

const ChartWrapper = styled.div`
  height: 400px;
  width: 100%;
`;

const LoadingChart = styled.div`
  height: 400px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #f8f9fa;
  border-radius: 8px;
  color: #666;
`;

const ErrorChart = styled.div`
  height: 400px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #f8f9fa;
  border-radius: 8px;
  color: #EA3943;
`;

const timeRanges = [
  { key: 'week', label: '7D', limit: 7 * 24, aggregate: 1, allData: false },
  { key: 'month', label: '1M', limit: 30 * 24, aggregate: 2, allData: false },
  { key: 'halfyear', label: '6M', limit: 180, aggregate: 1, allData: false },
  { key: 'year', label: '1Y', limit: 365, aggregate: 1, allData: false },
  { key: 'all', label: 'ALL', limit: 1, aggregate: 12, allData: true }
];

function TokenDetails({ token, onClose }) {
  const [selectedTimeRange, setSelectedTimeRange] = useState('week');
  const [apiRequestCount, setApiRequestCount] = useState(getApiRequestCounter());
  const [cryptoCompareRequestCount, setCryptoCompareRequestCount] = useState(getCryptoCompareRequestCounter());
  const [localProxyRequestCount, setLocalProxyRequestCount] = useState(getLocalProxyRequestCounter());
  
  // Subscribe to API request counter changes
  useEffect(() => {
    const unsubscribeCoinGecko = subscribeToApiCounter(setApiRequestCount);
    const unsubscribeCryptoCompare = subscribeToCryptoCompareCounter(setCryptoCompareRequestCount);
    const unsubscribeLocalProxy = subscribeToLocalProxyCounter(setLocalProxyRequestCount);
    return () => {
      unsubscribeCoinGecko();
      unsubscribeCryptoCompare();
      unsubscribeLocalProxy();
    };
  }, []);
  
  // Use hooks to get historical data from all three APIs
  const { data: coinGeckoData, isLoading: isLoadingCoinGecko, error: coinGeckoError } = useTokenHistory(token.id, selectedTimeRange);
  const { data: cryptoCompareData, isLoading: isLoadingCryptoCompare, error: cryptoCompareError } = useCryptoCompareHistory(token.symbol, selectedTimeRange);
  const { data: localProxyData, isLoading: isLoadingLocalProxy, error: localProxyError } = useLocalProxyHistory(token.id, selectedTimeRange);

  const formatNumber = (num) => {
    if (!num && num !== 0) return '—';
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(num);
  };

  const formatPercentage = (value) => {
    if (!value && value !== 0) return '—';
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
  };

  const handleTimeRangeChange = (timeRange) => {
    setSelectedTimeRange(timeRange);
  };

  const handleOverlayClick = (e) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  // Extract token data depending on the source
  const {
    id,
    name,
    symbol,
    image,
    current_price,
    market_cap,
    total_volume,
    price_change_percentage_24h,
    high_24h,
    low_24h,
    circulating_supply,
    total_supply
  } = token;

  return (
    <Container onClick={handleOverlayClick}>
      <Modal>
        <CloseButton onClick={onClose}>×</CloseButton>
        
        <TokenHeader>
          <TokenImage src={image} alt={name} />
          <TokenInfo>
            <TokenName>{name}</TokenName>
            <TokenSymbol>{symbol}</TokenSymbol>
          </TokenInfo>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginLeft: 'auto' }}>
            <ApiCounter>
              CoinGecko: {apiRequestCount}
            </ApiCounter>
            <ApiCounter>
              CryptoCompare: {cryptoCompareRequestCount}
            </ApiCounter>
            <ApiCounter>
              Local Proxy: {localProxyRequestCount}
            </ApiCounter>
          </div>
        </TokenHeader>

        <PriceInfo>
          <PriceCard>
            <PriceLabel>Current Price</PriceLabel>
            <PriceValue>{formatNumber(current_price)}</PriceValue>
          </PriceCard>
          
          <PriceCard>
            <PriceLabel>24h Change</PriceLabel>
            <PriceValue 
              $isPositive={price_change_percentage_24h >= 0}
              $isNegative={price_change_percentage_24h < 0}
            >
              {formatPercentage(price_change_percentage_24h)}
            </PriceValue>
          </PriceCard>
          
          <PriceCard>
            <PriceLabel>Market Cap</PriceLabel>
            <PriceValue>{formatNumber(market_cap)}</PriceValue>
          </PriceCard>
          
          <PriceCard>
            <PriceLabel>24h Volume</PriceLabel>
            <PriceValue>{formatNumber(total_volume)}</PriceValue>
          </PriceCard>
          
          <PriceCard>
            <PriceLabel>24h High</PriceLabel>
            <PriceValue>{formatNumber(high_24h)}</PriceValue>
          </PriceCard>
          
          <PriceCard>
            <PriceLabel>24h Low</PriceLabel>
            <PriceValue>{formatNumber(low_24h)}</PriceValue>
          </PriceCard>
        </PriceInfo>

        <ChartsContainer>
          <div style={{ textAlign: 'center', marginBottom: '16px' }}>
            <ChartTitle>Price History Comparison</ChartTitle>
            <TimeRangeSelector>
              {timeRanges.map(range => (
                <TimeRangeButton
                  key={range.key}
                  $active={selectedTimeRange === range.key}
                  onClick={() => handleTimeRangeChange(range.key)}
                >
                  {range.label}
                </TimeRangeButton>
              ))}
            </TimeRangeSelector>
          </div>

          <ChartsGrid>
            {/* CoinGecko Chart (Direct API) */}
            <ChartContainer>
              <ChartTitle style={{ fontSize: '16px', color: '#3861FB' }}>CoinGecko (Direct API)</ChartTitle>
              <ChartWrapper>
                {isLoadingCoinGecko ? (
                  <LoadingChart>Loading CoinGecko data...</LoadingChart>
                ) : coinGeckoError ? (
                  <ErrorChart>{coinGeckoError}</ErrorChart>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={coinGeckoData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis 
                        dataKey="date"
                        tick={{ fontSize: 10 }}
                      />
                      <YAxis 
                        domain={['dataMin', 'dataMax']}
                        tick={{ fontSize: 10 }}
                        tickFormatter={(value) => `$${value.toFixed(2)}`}
                      />
                      <Tooltip 
                        formatter={(value) => [`$${value.toFixed(2)}`, 'Price']}
                        labelFormatter={(label) => `Date: ${label}`}
                      />
                      <Line 
                        type="monotone" 
                        dataKey="price" 
                        stroke="#3861FB" 
                        strokeWidth={2}
                        dot={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </ChartWrapper>
            </ChartContainer>

            {/* CoinGecko Chart (Local Proxy) */}
            <ChartContainer>
              <ChartTitle style={{ fontSize: '16px', color: '#16C784' }}>CoinGecko (Local Proxy)</ChartTitle>
              <ChartWrapper>
                {isLoadingLocalProxy ? (
                  <LoadingChart>Loading Local Proxy data...</LoadingChart>
                ) : localProxyError ? (
                  <ErrorChart>{localProxyError}</ErrorChart>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={localProxyData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis 
                        dataKey="date"
                        tick={{ fontSize: 10 }}
                      />
                      <YAxis 
                        domain={['dataMin', 'dataMax']}
                        tick={{ fontSize: 10 }}
                        tickFormatter={(value) => `$${value.toFixed(2)}`}
                      />
                      <Tooltip 
                        formatter={(value) => [`$${value.toFixed(2)}`, 'Price']}
                        labelFormatter={(label) => `Date: ${label}`}
                      />
                      <Line 
                        type="monotone" 
                        dataKey="price" 
                        stroke="#16C784" 
                        strokeWidth={2}
                        dot={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </ChartWrapper>
            </ChartContainer>

            {/* CryptoCompare Chart */}
            <ChartContainer>
              <ChartTitle style={{ fontSize: '16px', color: '#FF6B35' }}>CryptoCompare Data</ChartTitle>
              <ChartWrapper>
                {isLoadingCryptoCompare ? (
                  <LoadingChart>Loading CryptoCompare data...</LoadingChart>
                ) : cryptoCompareError ? (
                  <ErrorChart>{cryptoCompareError}</ErrorChart>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={cryptoCompareData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis 
                        dataKey="date"
                        tick={{ fontSize: 10 }}
                      />
                      <YAxis 
                        domain={['dataMin', 'dataMax']}
                        tick={{ fontSize: 10 }}
                        tickFormatter={(value) => `$${value.toFixed(2)}`}
                      />
                      <Tooltip 
                        formatter={(value) => [`$${value.toFixed(2)}`, 'Price']}
                        labelFormatter={(label) => `Date: ${label}`}
                      />
                      <Line 
                        type="monotone" 
                        dataKey="price" 
                        stroke="#FF6B35" 
                        strokeWidth={2}
                        dot={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </ChartWrapper>
            </ChartContainer>
          </ChartsGrid>
        </ChartsContainer>
      </Modal>
    </Container>
  );
}

export default TokenDetails; 