import React, { useState, useEffect, useRef } from 'react';
import styled from 'styled-components';
import { proxyFetch, extractEndpointFromUrl } from '../utils/proxy_request';

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

const FileInput = styled.input`
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
`;

const Select = styled.select`
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  background: white;
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

const Checkbox = styled.input`
  margin: 0;
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

const PlayButton = styled.button`
  background: #10b981;
  color: white;
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  transition: all 0.2s ease;

  &:hover {
    background: #059669;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

const ClickableRow = styled(TableRow)`
  cursor: pointer;
  
  &:hover {
    background: #f9fafb;
  }
`;

const NonClickableCell = styled(TableCell)`
  position: relative;
`;

const RequestReplay = ({ onBack }) => {
  const [requests, setRequests] = useState([]);
  const [selectedRequests, setSelectedRequests] = useState(new Set());
  const [runMode, setRunMode] = useState('sequential');
  const [isPlaying, setIsPlaying] = useState(false);
  const [requestStatus, setRequestStatus] = useState({});
  const fileInputRef = useRef(null);

  // Load default requests.ndjson file on mount
  useEffect(() => {
    loadRequestsFile('/requests.ndjson');
  }, []);

  const loadRequestsFile = async (filePath) => {
    try {
      const response = await fetch(filePath);
      const text = await response.text();
      parseRequestsFile(text);
    } catch (error) {
      console.error('Error loading requests file:', error);
    }
  };

  const parseRequestsFile = (text) => {
    const lines = text.split('\n').filter(line => line.trim());
    const parsedRequests = lines.map((line, index) => {
      try {
        const request = JSON.parse(line);
        return {
          id: index,
          timestamp: request.ts,
          delta: request.delta,
          method: request.method,
          url: request.url,
          status: request.status,
          duration: request.ms,
          endpoint: extractEndpoint(request.url),
          idCount: extractIdCount(request.url)
        };
      } catch (error) {
        console.error('Error parsing line:', line, error);
        return null;
      }
    }).filter(Boolean);
    
    setRequests(parsedRequests);
    setSelectedRequests(new Set());
    setRequestStatus({});
  };

  const extractEndpoint = (url) => {
    // Extract endpoint without parameters
    const urlObj = new URL(url, 'http://localhost');
    return urlObj.pathname;
  };

  const extractIdCount = (url) => {
    // Extract ID count from URL parameters
    const urlObj = new URL(url, 'http://localhost');
    const ids = urlObj.searchParams.get('ids');
    if (ids) {
      return ids.split(',').length;
    }
    return 0;
  };

  const handleFileUpload = (event) => {
    const file = event.target.files[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        parseRequestsFile(e.target.result);
      };
      reader.readAsText(file);
    }
  };

  const toggleSelectAll = () => {
    if (selectedRequests.size === requests.length) {
      setSelectedRequests(new Set());
    } else {
      setSelectedRequests(new Set(requests.map(r => r.id)));
    }
  };

  const toggleRequestSelection = (requestId) => {
    const newSelection = new Set(selectedRequests);
    if (newSelection.has(requestId)) {
      newSelection.delete(requestId);
    } else {
      newSelection.add(requestId);
    }
    setSelectedRequests(newSelection);
  };

  const executeRequest = async (request) => {
    const startTime = Date.now();
    
    try {
      // Update status to running
      setRequestStatus(prev => ({
        ...prev,
        [request.id]: { status: 'running', progress: 0, startTime }
      }));

      // Simulate progress updates
      const progressInterval = setInterval(() => {
        setRequestStatus(prev => ({
          ...prev,
          [request.id]: { 
            ...prev[request.id], 
            progress: Math.min((Date.now() - startTime) / 50, 95) 
          }
        }));
      }, 50);

      // Extract endpoint from the original URL and make request through proxy
      const endpoint = extractEndpointFromUrl(request.url);
      const response = await proxyFetch(endpoint);
      const responseTime = Date.now() - startTime;
      
      clearInterval(progressInterval);

      // Check if response is from cache
      const isCached = response.headers.get('x-cache-status') === 'HIT' || 
                      response.headers.get('cache-control') || 
                      response.status === 304;
      const cacheStatus = response.headers.get('cache-status');

      // Check response data count
      let responseCount = 0;
      if (response.ok) {
        const data = await response.json();
        if (Array.isArray(data)) {
          responseCount = data.length;
        } else if (data && typeof data === 'object') {
          responseCount = Object.keys(data).length;
        }
      }

      setRequestStatus(prev => ({
        ...prev,
        [request.id]: {
          status: response.ok ? 'completed' : 'error',
          progress: 100,
          responseTime,
          responseCount,
          expectedCount: request.idCount,
          isCached,
          cacheStatus,
          statusCode: response.status
        }
      }));

    } catch (error) {
      const responseTime = Date.now() - startTime;
      setRequestStatus(prev => ({
        ...prev,
        [request.id]: {
          status: 'error',
          progress: 100,
          responseTime,
          error: error.message
        }
      }));
    }
  };

  const playSingleRequest = async (request) => {
    // Reset status for this request
    setRequestStatus(prev => ({
      ...prev,
      [request.id]: { status: 'pending', progress: 0 }
    }));

    await executeRequest(request);
  };

  const playRequests = async () => {
    if (selectedRequests.size === 0) return;

    setIsPlaying(true);
    const selectedRequestList = requests.filter(r => selectedRequests.has(r.id));

    // Reset status for selected requests
    const resetStatus = {};
    selectedRequestList.forEach(req => {
      resetStatus[req.id] = { status: 'pending', progress: 0 };
    });
    setRequestStatus(resetStatus);

    try {
      if (runMode === 'simultaneous') {
        // Run all requests simultaneously
        await Promise.all(selectedRequestList.map(executeRequest));
      } else {
        // Run requests sequentially
        for (let i = 0; i < selectedRequestList.length; i++) {
          const request = selectedRequestList[i];
          await executeRequest(request);
          
          if (i < selectedRequestList.length - 1) {
            // Calculate delay based on original timing or max 1 second
            let delay = 0;
            if (runMode === 'sequential') {
              const nextRequest = selectedRequestList[i + 1];
              delay = Math.abs(nextRequest.delta);
            } else if (runMode === 'sequential-limited') {
              const nextRequest = selectedRequestList[i + 1];
              delay = Math.min(Math.abs(nextRequest.delta), 1000);
            }
            
            if (delay > 0) {
              await new Promise(resolve => setTimeout(resolve, delay));
            }
          }
        }
      }
    } finally {
      setIsPlaying(false);
    }
  };

  const handleRowClick = (requestId, event) => {
    // Don't toggle if clicking on checkbox or play button
    if (event.target.type === 'checkbox' || event.target.closest('button')) {
      return;
    }
    toggleRequestSelection(requestId);
  };

  const formatDuration = (ms) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const getProgressInfo = (request) => {
    const status = requestStatus[request.id];
    if (!status) return { progress: 0, text: 'Pending' };

    switch (status.status) {
      case 'running':
        return { progress: status.progress, text: 'Running...' };
      case 'completed':
        const countMatch = status.expectedCount === status.responseCount;
        const cacheInfo = status.isCached ? ' (cached)' : '';
        const cacheStatusInfo = status.cacheStatus ? ` [${status.cacheStatus}]` : '';
        return { 
          progress: 100, 
          text: `${formatDuration(status.responseTime)}${cacheInfo}${cacheStatusInfo} - ${status.responseCount}/${status.expectedCount} ${countMatch ? '✓' : '⚠️'}` 
        };
      case 'error':
        return { progress: 100, text: `Error: ${status.error || 'Request failed'}` };
      default:
        return { progress: 0, text: 'Pending' };
    }
  };

  return (
    <Container>
      <Header>
        <Title>Requests Replay</Title>
        <BackButton onClick={onBack}>← Back to Main</BackButton>
      </Header>

      <ControlsSection>
        <ControlRow>
          <label>Load File:</label>
          <FileInput
            type="file"
            accept=".ndjson,.json,.txt"
            onChange={handleFileUpload}
            ref={fileInputRef}
          />
          <Button onClick={() => loadRequestsFile('/requests.ndjson')}>
            Load Default (requests.ndjson)
          </Button>
        </ControlRow>

        <ControlRow>
          <label>Run Mode:</label>
          <Select value={runMode} onChange={(e) => setRunMode(e.target.value)}>
            <option value="sequential">Sequential (original timing)</option>
            <option value="sequential-limited">Sequential (max 1 sec delay)</option>
            <option value="simultaneous">Simultaneous</option>
          </Select>
          
          <Button
            variant="primary"
            onClick={playRequests}
            disabled={isPlaying || selectedRequests.size === 0}
          >
            {isPlaying ? 'Playing...' : `Play (${selectedRequests.size} selected)`}
          </Button>

          <Button onClick={toggleSelectAll}>
            {selectedRequests.size === requests.length ? 'Deselect All' : 'Select All'}
          </Button>
        </ControlRow>
      </ControlsSection>

      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHeaderCell>Select</TableHeaderCell>
              <TableHeaderCell>Play</TableHeaderCell>
              <TableHeaderCell>Time Offset</TableHeaderCell>
              <TableHeaderCell>Endpoint</TableHeaderCell>
              <TableHeaderCell>ID Count</TableHeaderCell>
              <TableHeaderCell>Progress</TableHeaderCell>
            </TableRow>
          </TableHeader>
          <tbody>
            {requests.map((request) => {
              const progressInfo = getProgressInfo(request);
              const status = requestStatus[request.id];
              const isRequestRunning = status?.status === 'running';
              
              return (
                <ClickableRow 
                  key={request.id}
                  onClick={(e) => handleRowClick(request.id, e)}
                >
                  <NonClickableCell>
                    <Checkbox
                      type="checkbox"
                      checked={selectedRequests.has(request.id)}
                      onChange={() => toggleRequestSelection(request.id)}
                    />
                  </NonClickableCell>
                  <NonClickableCell>
                    <PlayButton
                      onClick={(e) => {
                        e.stopPropagation();
                        playSingleRequest(request);
                      }}
                      disabled={isRequestRunning}
                      title="Play this request"
                    >
                      ▶
                    </PlayButton>
                  </NonClickableCell>
                  <TableCell>{formatDuration(Math.abs(request.delta))}</TableCell>
                  <TableCell>{request.endpoint}</TableCell>
                  <TableCell>{request.idCount}</TableCell>
                  <TableCell>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <ProgressBar style={{ width: '120px' }}>
                        <ProgressFill 
                          progress={progressInfo.progress} 
                          status={status?.status}
                        />
                      </ProgressBar>
                      <span style={{ fontSize: '12px', minWidth: '200px' }}>
                        {progressInfo.text}
                      </span>
                      {status?.status && (
                        <StatusBadge status={status.status}>
                          {status.status}
                        </StatusBadge>
                      )}
                    </div>
                  </TableCell>
                </ClickableRow>
              );
            })}
          </tbody>
        </Table>
      </TableContainer>

      {requests.length === 0 && (
        <div style={{ textAlign: 'center', padding: '40px', color: '#6b7280' }}>
          No requests loaded. Upload a file or load the default requests.ndjson file.
        </div>
      )}
    </Container>
  );
};

export default RequestReplay;