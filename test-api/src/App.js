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
  } = useCoinGeckoPriceData();

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
      {/* Show appropriate stats based on active main tab */}
      {(
        <>
          <Stats stats={coinGeckoStats} title="CoinGecko Data Stats" />
          <Stats stats={coinGeckoPriceStats} title="CoinGecko Price Data Stats" />
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
        />
      </ErrorBoundary>
    </Layout>
  );
}

export default App; 