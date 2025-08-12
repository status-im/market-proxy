import React, { useState, useRef } from 'react';
import styled from 'styled-components';
import { proxyFetch } from '../utils/proxy_request';

const Container = styled.div`
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
`;

const Header = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e5e7eb;
`;

const Title = styled.h1`
  font-size: 32px;
  font-weight: 600;
  color: #09101c;
  margin: 0;
`;

const BackButton = styled.button`
  background: #f3f4f6;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: #e5e7eb;
  }
`;

const ControlsSection = styled.div`
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
`;

const ControlRow = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;

  &:last-child {
    margin-bottom: 0;
  }
`;

const Input = styled.input`
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  width: 120px;
`;

const Button = styled.button`
  background: ${props => props.variant === 'primary' ? '#6366f1' : '#f3f4f6'};
  color: ${props => props.variant === 'primary' ? 'white' : '#374151'};
  border: 1px solid ${props => props.variant === 'primary' ? '#6366f1' : '#d1d5db'};
  border-radius: 8px;
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  
  &:hover {
    background: ${props => props.variant === 'primary' ? '#5856eb' : '#e5e7eb'};
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

const TableContainer = styled.div`
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 24px;
`;

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
`;

const TableHeader = styled.thead`
  background: #f9fafb;
`;

const TableRow = styled.tr`
  border-bottom: 1px solid #e5e7eb;

  &:hover {
    background: #f9fafb;
  }
`;

const TableHeaderCell = styled.th`
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: #374151;
  font-size: 14px;
`;

const TableCell = styled.td`
  padding: 12px 16px;
  font-size: 14px;
  color: #374151;
`;

const PlayButton = styled.button`
  background: ${props => props.disabled ? '#9ca3af' : '#10b981'};
  color: white;
  border: none;
  border-radius: 4px;
  padding: 6px 12px;
  font-size: 12px;
  cursor: ${props => props.disabled ? 'not-allowed' : 'pointer'};
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 60px;
  height: 28px;
  transition: all 0.2s ease;

  &:hover:not(:disabled) {
    background: #059669;
  }
`;

const ProgressBar = styled.div`
  width: 100%;
  height: 20px;
  background: #e5e7eb;
  border-radius: 10px;
  overflow: hidden;
`;

const ProgressFill = styled.div`
  height: 100%;
  background: ${props => props.status === 'completed' ? '#10b981' : props.status === 'error' ? '#ef4444' : '#6366f1'};
  width: ${props => props.progress}%;
  transition: width 0.3s ease;
`;

const StatusBadge = styled.span`
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  
  ${props => props.status === 'completed' && `
    background: #dcfce7;
    color: #166534;
  `}
  
  ${props => props.status === 'error' && `
    background: #fee2e2;
    color: #991b1b;
  `}
  
  ${props => props.status === 'running' && `
    background: #dbeafe;
    color: #1e40af;
  `}
  
  ${props => props.status === 'pending' && `
    background: #f3f4f6;
    color: #374151;
  `}
`;

const LogSection = styled.div`
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 24px;
`;

const LogTitle = styled.h3`
  font-size: 18px;
  font-weight: 600;
  color: #374151;
  margin: 0 0 16px 0;
`;

const LogEntry = styled.div`
  padding: 12px;
  margin-bottom: 8px;
  background: #f9fafb;
  border-radius: 8px;
  font-family: 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  border-left: 4px solid ${props => props.type === 'error' ? '#ef4444' : props.type === 'success' ? '#10b981' : '#6366f1'};
`;

const EndpointTester = ({ onBack }) => {
  const [marketPages, setMarketPages] = useState(40);
  const [isRunningAll, setIsRunningAll] = useState(false);
  const [endpointStatus, setEndpointStatus] = useState({});
  const [logs, setLogs] = useState([]);
  const [globalData, setGlobalData] = useState({
    coinsList: null,
    marketsData: null
  });

  // Define all endpoints from server.go
  const endpoints = [
    {
      id: 'coins-list',
      name: 'Coins List',
      path: '/v1/coins/list',
      description: 'Get all coins list',
      dependencies: []
    },
    {
      id: 'coins-markets',
      name: 'Coins Markets',
      path: '/v1/coins/markets',
      description: 'Get market data for coins (paginated)',
      dependencies: []
    },
    {
      id: 'leaderboard-prices',
      name: 'Leaderboard Prices',
      path: '/v1/leaderboard/prices',
      description: 'Get leaderboard prices',
      dependencies: []
    },
    {
      id: 'leaderboard-simple-prices',
      name: 'Leaderboard Simple Prices',
      path: '/v1/leaderboard/simpleprices',
      description: 'Get simple prices for leaderboard',
      dependencies: []
    },
    {
      id: 'leaderboard-markets',
      name: 'Leaderboard Markets',
      path: '/v1/leaderboard/markets',
      description: 'Get leaderboard markets',
      dependencies: []
    },
    {
      id: 'simple-price-markets',
      name: 'Simple Price for Markets',
      path: '/v1/simple/price',
      description: 'Get prices for market IDs from coins/markets',
      dependencies: ['coins-markets']
    },
    {
      id: 'simple-price-coins',
      name: 'Simple Price for Coins',
      path: '/v1/simple/price',
      description: 'Get prices for coin IDs from coins/list',
      dependencies: ['coins-list']
    },
    {
      id: 'asset-platforms',
      name: 'Asset Platforms',
      path: '/v1/asset_platforms',
      description: 'Get asset platforms',
      dependencies: []
    }
  ];

  const addLog = (message, type = 'info') => {
    const timestamp = new Date().toLocaleTimeString();
    setLogs(prev => [...prev, { timestamp, message, type }]);
  };

  const updateEndpointStatus = (endpointId, status) => {
    setEndpointStatus(prev => ({
      ...prev,
      [endpointId]: { ...prev[endpointId], ...status }
    }));
  };

  const formatDuration = (ms) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const executeCoinsListEndpoint = async () => {
    const startTime = Date.now();
    updateEndpointStatus('coins-list', { status: 'running', progress: 50 });
    
    try {
      const response = await proxyFetch('/v1/coins/list');
      const data = await response.json();
      const duration = Date.now() - startTime;
      
      if (response.ok) {
        // Analyze platforms
        const platformCounts = {};
        let totalTokens = 0;
        
        data.forEach(coin => {
          totalTokens++;
          Object.keys(coin.platforms || {}).forEach(platform => {
            if (coin.platforms[platform]) {
              platformCounts[platform] = (platformCounts[platform] || 0) + 1;
            }
          });
        });
        
        const uniquePlatforms = Object.keys(platformCounts).length;
        
        setGlobalData(prev => ({ ...prev, coinsList: data }));
        updateEndpointStatus('coins-list', { 
          status: 'completed', 
          progress: 100, 
          responseTime: duration,
          recordCount: totalTokens 
        });
        
        addLog(`coins/list - ${totalTokens} tokens, ${uniquePlatforms} platforms`, 'success');
        Object.entries(platformCounts).forEach(([platform, count]) => {
          addLog(`  ${platform}: ${count} tokens`, 'info');
        });
        
      } else {
        throw new Error(`HTTP ${response.status}`);
      }
    } catch (error) {
      const duration = Date.now() - startTime;
      updateEndpointStatus('coins-list', { 
        status: 'error', 
        progress: 100, 
        responseTime: duration,
        error: error.message 
      });
      addLog(`coins/list - Error: ${error.message}`, 'error');
    }
  };

  const executeCoinsMarketsEndpoint = async () => {
    const startTime = Date.now();
    updateEndpointStatus('coins-markets', { status: 'running', progress: 0 });
    
    try {
      let allMarkets = [];
      
      for (let page = 1; page <= marketPages; page++) {
        updateEndpointStatus('coins-markets', { 
          status: 'running', 
          progress: (page / marketPages) * 100 
        });
        
        addLog(`coins/markets - Fetching page ${page}/${marketPages}`, 'info');
        
        const response = await proxyFetch(`/v1/coins/markets?page=${page}&per_page=250`);
        const pageData = await response.json();
        
        if (response.ok && Array.isArray(pageData) && pageData.length > 0) {
          allMarkets = [...allMarkets, ...pageData];
          addLog(`  Page ${page}: ${pageData.length} tokens (total: ${allMarkets.length})`, 'info');
        } else {
          break; // No more data
        }
      }
      
      const duration = Date.now() - startTime;
      setGlobalData(prev => ({ ...prev, marketsData: allMarkets }));
      updateEndpointStatus('coins-markets', { 
        status: 'completed', 
        progress: 100, 
        responseTime: duration,
        recordCount: allMarkets.length 
      });
      
      addLog(`coins/markets - Total: ${allMarkets.length} tokens from ${marketPages} pages`, 'success');
      
    } catch (error) {
      const duration = Date.now() - startTime;
      updateEndpointStatus('coins-markets', { 
        status: 'error', 
        progress: 100, 
        responseTime: duration,
        error: error.message 
      });
      addLog(`coins/markets - Error: ${error.message}`, 'error');
    }
  };

  const executeSimpleEndpoint = async (endpoint, path, description) => {
    const startTime = Date.now();
    updateEndpointStatus(endpoint, { status: 'running', progress: 50 });
    
    try {
      const response = await proxyFetch(path);
      const data = await response.json();
      const duration = Date.now() - startTime;
      
      if (response.ok) {
        const recordCount = Array.isArray(data) ? data.length : Object.keys(data).length;
        updateEndpointStatus(endpoint, { 
          status: 'completed', 
          progress: 100, 
          responseTime: duration,
          recordCount 
        });
        addLog(`${description} - ${recordCount} records`, 'success');
      } else {
        throw new Error(`HTTP ${response.status}`);
      }
    } catch (error) {
      const duration = Date.now() - startTime;
      updateEndpointStatus(endpoint, { 
        status: 'error', 
        progress: 100, 
        responseTime: duration,
        error: error.message 
      });
      addLog(`${description} - Error: ${error.message}`, 'error');
    }
  };

  const executeSimplePriceEndpoint = async (endpointId, sourceData, sourceType) => {
    if (!sourceData) {
      addLog(`${endpointId} - No ${sourceType} data available. Run ${sourceType} first.`, 'error');
      return;
    }

    const startTime = Date.now();
    updateEndpointStatus(endpointId, { status: 'running', progress: 0 });
    
    try {
      // Get all IDs from source data
      let allIds = [];
      if (sourceType === 'coins/list') {
        allIds = sourceData.map(coin => coin.id);
      } else if (sourceType === 'coins/markets') {
        allIds = sourceData.map(coin => coin.id);
      }
      
      addLog(`simple/price for ${sourceType} - Processing ${allIds.length} IDs in chunks of 500`, 'info');
      
      // Split into chunks of 500
      const chunkSize = 500;
      const chunks = [];
      for (let i = 0; i < allIds.length; i += chunkSize) {
        chunks.push(allIds.slice(i, i + chunkSize));
      }
      
      let allPriceData = {};
      let totalReceived = 0;
      
      // Process each chunk
      for (let i = 0; i < chunks.length; i++) {
        const chunk = chunks[i];
        const progress = ((i + 1) / chunks.length) * 100;
        
        updateEndpointStatus(endpointId, { 
          status: 'running', 
          progress: Math.min(progress, 95) 
        });
        
        addLog(`  Chunk ${i + 1}/${chunks.length}: ${chunk.length} IDs`, 'info');
        
        const idsParam = chunk.join(',');
        const response = await proxyFetch(`/v1/simple/price?ids=${idsParam}&vs_currencies=usd`);
        
        if (response.ok) {
          const chunkData = await response.json();
          const chunkCount = Object.keys(chunkData).length;
          allPriceData = { ...allPriceData, ...chunkData };
          totalReceived += chunkCount;
          
          addLog(`    Received: ${chunkCount}/${chunk.length} prices`, 'info');
        } else {
          throw new Error(`HTTP ${response.status} on chunk ${i + 1}`);
        }
        
        // Small delay between chunks
        if (i < chunks.length - 1) {
          await new Promise(resolve => setTimeout(resolve, 200));
        }
      }
      
      const duration = Date.now() - startTime;
      const isComplete = totalReceived === allIds.length;
      const completionEmoji = isComplete ? '✅' : '⚠️';
      
      updateEndpointStatus(endpointId, { 
        status: 'completed', 
        progress: 100, 
        responseTime: duration,
        recordCount: totalReceived,
        requestedCount: allIds.length,
        isComplete 
      });
      
      addLog(`simple/price for ${sourceType} - ${totalReceived}/${allIds.length} prices received ${completionEmoji}`, 
             isComplete ? 'success' : 'info');
      
    } catch (error) {
      const duration = Date.now() - startTime;
      updateEndpointStatus(endpointId, { 
        status: 'error', 
        progress: 100, 
        responseTime: duration,
        error: error.message 
      });
      addLog(`simple/price for ${sourceType} - Error: ${error.message}`, 'error');
    }
  };

  const executeSingleEndpoint = async (endpoint) => {
    addLog(`Starting ${endpoint.name}...`, 'info');
    
    switch (endpoint.id) {
      case 'coins-list':
        await executeCoinsListEndpoint();
        break;
      case 'coins-markets':
        await executeCoinsMarketsEndpoint();
        break;
      case 'leaderboard-prices':
        await executeSimpleEndpoint(endpoint.id, endpoint.path, 'leaderboard/prices');
        break;
      case 'leaderboard-simple-prices':
        await executeSimpleEndpoint(endpoint.id, endpoint.path, 'leaderboard/simpleprices');
        break;
      case 'leaderboard-markets':
        await executeSimpleEndpoint(endpoint.id, endpoint.path, 'leaderboard/markets');
        break;
      case 'simple-price-markets':
        await executeSimplePriceEndpoint(endpoint.id, globalData.marketsData, 'coins/markets');
        break;
      case 'simple-price-coins':
        await executeSimplePriceEndpoint(endpoint.id, globalData.coinsList, 'coins/list');
        break;
      case 'asset-platforms':
        await executeSimpleEndpoint(endpoint.id, endpoint.path, 'asset_platforms');
        break;
    }
  };

  const runAllEndpoints = async () => {
    setIsRunningAll(true);
    setLogs([]);
    setEndpointStatus({});
    
    addLog('Starting full endpoint test...', 'info');
    
    // Run endpoints in order, respecting dependencies
    const orderedEndpoints = [
      'coins-list',
      'coins-markets', 
      'leaderboard-prices',
      'leaderboard-simple-prices',
      'leaderboard-markets',
      'asset-platforms',
      'simple-price-markets',
      'simple-price-coins'
    ];
    
    for (const endpointId of orderedEndpoints) {
      const endpoint = endpoints.find(ep => ep.id === endpointId);
      if (endpoint) {
        await executeSingleEndpoint(endpoint);
        // Small delay between requests
        await new Promise(resolve => setTimeout(resolve, 500));
      }
    }
    
    addLog('All endpoints test completed!', 'success');
    setIsRunningAll(false);
  };

  const getProgressInfo = (endpoint) => {
    const status = endpointStatus[endpoint.id];
    if (!status) return { progress: 0, text: 'Pending', status: 'pending' };

    switch (status.status) {
      case 'running':
        return { progress: status.progress, text: 'Running...', status: 'running' };
      case 'completed':
        let resultText = `${formatDuration(status.responseTime)} - ${status.recordCount} records`;
        
        // Add emoji indicators for simple/price endpoints
        if (endpoint.id === 'simple-price-markets' || endpoint.id === 'simple-price-coins') {
          if (status.requestedCount !== undefined) {
            const completionEmoji = status.isComplete ? '✅' : '⚠️';
            resultText = `${formatDuration(status.responseTime)} - ${status.recordCount}/${status.requestedCount} ${completionEmoji}`;
          }
        }
        
        return { 
          progress: 100, 
          text: resultText,
          status: 'completed' 
        };
      case 'error':
        return { progress: 100, text: `Error: ${status.error}`, status: 'error' };
      default:
        return { progress: 0, text: 'Pending', status: 'pending' };
    }
  };

  const canRunEndpoint = (endpoint) => {
    if (endpoint.dependencies.length === 0) return true;
    
    return endpoint.dependencies.every(dep => {
      if (dep === 'coins-list') return globalData.coinsList !== null;
      if (dep === 'coins-markets') return globalData.marketsData !== null;
      return false;
    });
  };

  return (
    <Container>
      <Header>
        <Title>Endpoint Tester</Title>
        <BackButton onClick={onBack}>← Back to Main</BackButton>
      </Header>

      <ControlsSection>
        <ControlRow>
          <label>Market Pages to fetch:</label>
          <Input
            type="number"
            value={marketPages}
            onChange={(e) => setMarketPages(parseInt(e.target.value) || 1)}
            min="1"
            max="100"
          />
          <Button
            variant="primary"
            onClick={runAllEndpoints}
            disabled={isRunningAll}
          >
            {isRunningAll ? 'Running All...' : 'Run All Endpoints'}
          </Button>
        </ControlRow>
      </ControlsSection>

      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHeaderCell>Play</TableHeaderCell>
              <TableHeaderCell>Endpoint</TableHeaderCell>
              <TableHeaderCell>Description</TableHeaderCell>
              <TableHeaderCell>Progress</TableHeaderCell>
              <TableHeaderCell>Result</TableHeaderCell>
            </TableRow>
          </TableHeader>
          <tbody>
            {endpoints.map((endpoint) => {
              const progressInfo = getProgressInfo(endpoint);
              const canRun = canRunEndpoint(endpoint);
              const isRunning = progressInfo.status === 'running';
              
              return (
                <TableRow key={endpoint.id}>
                  <TableCell>
                    <PlayButton
                      onClick={() => executeSingleEndpoint(endpoint)}
                      disabled={!canRun || isRunning || isRunningAll}
                      title={!canRun ? `Requires: ${endpoint.dependencies.join(', ')}` : 'Run this endpoint'}
                    >
                      ▶
                    </PlayButton>
                  </TableCell>
                  <TableCell>
                    <strong>{endpoint.name}</strong>
                    <br />
                    <small style={{ color: '#6b7280' }}>{endpoint.path}</small>
                  </TableCell>
                  <TableCell>{endpoint.description}</TableCell>
                  <TableCell>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <ProgressBar style={{ width: '120px' }}>
                        <ProgressFill 
                          progress={progressInfo.progress} 
                          status={progressInfo.status}
                        />
                      </ProgressBar>
                      <StatusBadge status={progressInfo.status}>
                        {progressInfo.status}
                      </StatusBadge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span style={{ fontSize: '14px' }}>
                      {progressInfo.text}
                    </span>
                  </TableCell>
                </TableRow>
              );
            })}
          </tbody>
        </Table>
      </TableContainer>

      <LogSection>
        <LogTitle>Execution Log</LogTitle>
        {logs.length === 0 ? (
          <div style={{ color: '#6b7280', fontStyle: 'italic' }}>
            No logs yet. Run an endpoint to see detailed results.
          </div>
        ) : (
          logs.map((log, index) => (
            <LogEntry key={index} type={log.type}>
              <strong>[{log.timestamp}]</strong> {log.message}
            </LogEntry>
          ))
        )}
      </LogSection>
    </Container>
  );
};

export default EndpointTester;