import React, { useState } from 'react';
import ErrorBoundary from './ErrorBoundary';
import CryptoDataTable from './CryptoDataTable';
import Tabs from './Tabs';
import Stats from './Stats';
import TokenDetails from './TokenDetails';
import { Loading, Error } from './LoadingAndErrors';
import useCoinGeckoData from '../hooks/useCoinGeckoData';
import useCoinGeckoPriceData from '../hooks/useCoinGeckoPriceData';
import styled from 'styled-components';

const Container = styled.div`
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
`;

const BackButton = styled.button`
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: transparent;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  color: #647084;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 24px;
  transition: all 0.2s ease;

  &:hover {
    background: #f9fafb;
    border-color: #d1d5db;
  }
`;

const Header = styled.div`
  margin-bottom: 32px;
`;

const Title = styled.h1`
  font-size: 32px;
  font-weight: 600;
  color: #09101c;
  margin: 0 0 8px 0;
`;

const Description = styled.p`
  font-size: 16px;
  color: #647084;
  margin: 0;
`;

const ConfigSection = styled.div`
  margin-bottom: 20px;
  padding: 20px;
  background: #f8fafc;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
`;

const SectionTitle = styled.h3`
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
  color: #374151;
`;

const ButtonGroup = styled.div`
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
`;

const ToggleButton = styled.button`
  padding: 8px 16px;
  border: 1px solid;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s ease;
  
  ${props => props.active ? `
    background-color: ${props.color || '#6366f1'};
    border-color: ${props.color || '#6366f1'};
    color: white;
  ` : `
    background-color: white;
    border-color: ${props.color || '#6366f1'};
    color: ${props.color || '#6366f1'};
  `}

  &:hover {
    opacity: 0.8;
  }
`;

const ConfigDescription = styled.p`
  margin: 12px 0 0 0;
  font-size: 12px;
  color: #6b7280;
  line-height: 1.4;
`;

function Leaderboard({ onBack }) {
  // Sub tab state
  const [activeTab, setActiveTab] = useState('All');

  // Endpoint state for prices
  const [priceEndpoint, setPriceEndpoint] = useState('prices');

  // Endpoint state for token data
  const [tokenEndpoint, setTokenEndpoint] = useState('leaderboard');

  // Selected token state
  const [selectedToken, setSelectedToken] = useState(null);

  // CoinGecko data with token endpoint parameter
  const {
    coinGeckoData,
    isLoading: isLoadingCoinGecko,
    error: coinGeckoError,
    stats: coinGeckoStats
  } = useCoinGeckoData(tokenEndpoint);

  const {
    coinGeckoPriceData,
    error: coinGeckoPriceError,
    stats: coinGeckoPriceStats
  } = useCoinGeckoPriceData(priceEndpoint);

  // Main tabs
  const mainTabs = ['CoinGecko'];

  // Filter tabs for each data source
  const tabs = ['All', '🔥 Trending', 'New', 'Gainers', 'Losers', 'Meme', 'AI', 'Gaming', '⭐ Watchlist'];

  const isLoading = isLoadingCoinGecko && coinGeckoData.length === 0;
  const error = coinGeckoError || coinGeckoPriceError;

  // Handle token selection
  const handleTokenClick = (token) => {
    setSelectedToken(token);
  };

  const handleCloseTokenDetails = () => {
    setSelectedToken(null);
  };

  if (isLoading) {
    return (
      <Container>
        <Loading />
      </Container>
    );
  }

  if (error) {
    return (
      <Container>
        <Error message={error} />
      </Container>
    );
  }

  return (
    <Container>
      <BackButton onClick={onBack}>
        ← Back to Utilities
      </BackButton>

      <Header>
        <Title>Crypto Leaderboard</Title>
        <Description>
          Real-time cryptocurrency market data from CoinGecko with configurable data sources
        </Description>
      </Header>

      {/* Token data source switcher */}
      <ConfigSection>
        <SectionTitle>Token Data Source</SectionTitle>
        <ButtonGroup>
          <ToggleButton
            onClick={() => setTokenEndpoint('leaderboard')}
            active={tokenEndpoint === 'leaderboard'}
            color="#10b981"
          >
            Optimized Leaderboard
          </ToggleButton>
          <ToggleButton
            onClick={() => setTokenEndpoint('coins')}
            active={tokenEndpoint === 'coins'}
            color="#10b981"
          >
            Coins/Markets (250)
          </ToggleButton>
        </ButtonGroup>
        <ConfigDescription>
          {tokenEndpoint === 'leaderboard' 
            ? 'Using /v1/leaderboard/markets - optimized endpoint with curated token data'
            : 'Using /v1/coins/markets?per_page=250 - first 250 tokens from standard coins endpoint'
          }
        </ConfigDescription>
      </ConfigSection>

      {/* Price endpoint switcher */}
      <ConfigSection>
        <SectionTitle>Price Data Source</SectionTitle>
        <ButtonGroup>
          <ToggleButton
            onClick={() => setPriceEndpoint('prices')}
            active={priceEndpoint === 'prices'}
            color="#3b82f6"
          >
            By Symbol (Binance Format)
          </ToggleButton>
          <ToggleButton
            onClick={() => setPriceEndpoint('simpleprices')}
            active={priceEndpoint === 'simpleprices'}
            color="#3b82f6"
          >
            By Token ID (CoinGecko Format)
          </ToggleButton>
        </ButtonGroup>
        <ConfigDescription>
          {priceEndpoint === 'prices' 
            ? 'Using /v1/leaderboard/prices - returns prices by symbol (BTC, ETH, etc.)'
            : 'Using /v1/leaderboard/simpleprices - returns prices by token ID (bitcoin, ethereum, etc.)'
          }
        </ConfigDescription>
      </ConfigSection>

      {/* Show appropriate stats */}
      <>
        <Stats stats={coinGeckoStats} title={`CoinGecko Data Stats (${tokenEndpoint})`} />
        <Stats stats={coinGeckoPriceStats} title={`CoinGecko Price Data Stats (${priceEndpoint})`} />
      </>

      {/* Tabs navigation */}
      <Tabs
        mainTabs={mainTabs}
        activeMainTab="CoinGecko"
        onMainTabChange={() => {}}
        tabs={tabs}
        activeTab={activeTab}
        onTabChange={setActiveTab}
      />

      {/* Use shared table component for both data sources */}
      <ErrorBoundary>
        <CryptoDataTable
          data={coinGeckoData}
          priceData={coinGeckoPriceData}
          source="CoinGecko"
          priceEndpoint={priceEndpoint}
          onTokenClick={handleTokenClick}
        />
      </ErrorBoundary>

      {/* Token details modal */}
      {selectedToken && (
        <TokenDetails
          token={selectedToken}
          onClose={handleCloseTokenDetails}
        />
      )}
    </Container>
  );
}

export default Leaderboard;