import React, { useState } from 'react';
import Layout from './components/Layout';
import MainPage from './components/MainPage';
import Leaderboard from './components/Leaderboard';
import RequestReplay from './components/RequestReplay';
import EndpointTester from './components/EndpointTester';

function App() {
  const [currentPage, setCurrentPage] = useState('main');

  const handleNavigate = (page) => {
    setCurrentPage(page);
  };

  const handleBack = () => {
    setCurrentPage('main');
  };

  const renderCurrentPage = () => {
    switch (currentPage) {
      case 'leaderboard':
        return <Leaderboard onBack={handleBack} />;
      case 'requests-replay':
        return <RequestReplay onBack={handleBack} />;
      case 'endpoint-tester':
        return <EndpointTester onBack={handleBack} />;
      default:
        return <MainPage onNavigate={handleNavigate} />;
    }
  };

  return (
    <Layout>
      {renderCurrentPage()}
    </Layout>
  );
}

export default App; 