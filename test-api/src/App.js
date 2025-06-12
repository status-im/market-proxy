import React, { useState } from 'react';
import ErrorBoundary from './components/ErrorBoundary';
import CryptoDataTable from './components/CryptoDataTable';
import Layout from './components/Layout';
import Tabs from './components/Tabs';
import Stats from './components/Stats';
import { Loading, Error } from './components/LoadingAndErrors';
import useCoinGeckoData from './hooks/useCoinGeckoData';
import useCoinGeckoPriceData from './hooks/useCoinGeckoPriceData';

function App() {
  // Main tab state
  const [activeMainTab, setActiveMainTab] = useState('CoinGecko');

  // Sub tab state
  const [activeTab, setActiveTab] = useState('All');

  // Endpoint state for prices
  const [priceEndpoint, setPriceEndpoint] = useState('prices');

  // CoinGecko data
  const {
    coinGeckoData,
    isLoading: isLoadingCoinGecko,
    error: coinGeckoError,
    stats: coinGeckoStats
  } = useCoinGeckoData();

  const {
    coinGeckoPriceData,
    // isLoading: isLoadingCoinGeckoPrices, // Not used currently
    error: coinGeckoPriceError,
    stats: coinGeckoPriceStats
  } = useCoinGeckoPriceData(priceEndpoint);

  // Main tabs
  const mainTabs = ['CoinGecko'];

  // Filter tabs for each data source
  const tabs = ['All', '🔥 Trending', 'New', 'Gainers', 'Losers', 'Meme', 'AI', 'Gaming', '⭐ Watchlist'];

  const isLoading = (activeMainTab === 'CoinGecko' && isLoadingCoinGecko && coinGeckoData.length === 0);

  const error = (activeMainTab === 'CoinGecko' && (coinGeckoError || coinGeckoPriceError));

  // Get the current active data and price data based on active tab
  const getCurrentData = () => {
    if (activeMainTab === 'CoinGecko') {
      return {
        data: coinGeckoData,
        priceData: coinGeckoPriceData,
        source: 'CoinGecko'
      };
    } else {
      return {};
    }
  };

  if (isLoading) {
    return (
      <Layout title="Crypto Dashboard">
        <Loading />
      </Layout>
    );
  }

  if (error) {
    return (
      <Layout title="Crypto Dashboard">
        <Error message={error} />
      </Layout>
    );
  }

  const { data, priceData: activePrice, source } = getCurrentData();

  return (
    <Layout title="Crypto Dashboard">
      {/* Price endpoint switcher */}
      <div style={{ 
        marginBottom: '20px', 
        padding: '15px', 
        background: '#f5f5f5', 
        borderRadius: '8px',
        border: '1px solid #ddd'
      }}>
        <h3 style={{ margin: '0 0 10px 0', fontSize: '16px', color: '#333' }}>Price Data Source:</h3>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button
            onClick={() => setPriceEndpoint('prices')}
            style={{
              padding: '8px 16px',
              border: '1px solid #007bff',
              borderRadius: '4px',
              backgroundColor: priceEndpoint === 'prices' ? '#007bff' : '#fff',
              color: priceEndpoint === 'prices' ? '#fff' : '#007bff',
              cursor: 'pointer',
              fontSize: '14px'
            }}
          >
            By Symbol (Binance Format)
          </button>
          <button
            onClick={() => setPriceEndpoint('simpleprices')}
            style={{
              padding: '8px 16px',
              border: '1px solid #007bff',
              borderRadius: '4px',
              backgroundColor: priceEndpoint === 'simpleprices' ? '#007bff' : '#fff',
              color: priceEndpoint === 'simpleprices' ? '#fff' : '#007bff',
              cursor: 'pointer',
              fontSize: '14px'
            }}
          >
            By Token ID (CoinGecko Format)
          </button>
        </div>
        <p style={{ margin: '10px 0 0 0', fontSize: '12px', color: '#666' }}>
          {priceEndpoint === 'prices' 
            ? 'Using /v1/leaderboard/prices - returns prices by symbol (BTC, ETH, etc.)'
            : 'Using /v1/leaderboard/simpleprices - returns prices by token ID (bitcoin, ethereum, etc.)'
          }
        </p>
      </div>

      {/* Show appropriate stats based on active main tab */}
      {(
        <>
          <Stats stats={coinGeckoStats} title="CoinGecko Data Stats" />
          <Stats stats={coinGeckoPriceStats} title={`CoinGecko Price Data Stats (${priceEndpoint})`} />
        </>
      )}

      {/* Tabs navigation */}
      <Tabs
        mainTabs={mainTabs}
        activeMainTab={activeMainTab}
        onMainTabChange={setActiveMainTab}
        tabs={tabs}
        activeTab={activeTab}
        onTabChange={setActiveTab}
      />

      {/* Use shared table component for both data sources */}
      <ErrorBoundary>
        <CryptoDataTable
          data={data}
          priceData={activePrice}
          source={source}
          priceEndpoint={priceEndpoint}
        />
      </ErrorBoundary>
    </Layout>
  );
}

export default App; 