import React from 'react';
import styled from 'styled-components';

const StatsWrapper = styled.div`
  margin-bottom: 30px;
`;

const StatsTitle = styled.h3`
  margin-bottom: 10px;
  color: #1E2026;
  font-size: 18px;
  font-weight: 600;
`;

const StatsContainer = styled.div`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  padding: 20px;
  background-color: #f8f9fa;
  border-radius: 12px;
`;

const StatCard = styled.div`
  text-align: center;
`;

const StatValue = styled.div`
  font-size: 24px;
  font-weight: bold;
  color: #3861FB;
`;

const StatLabel = styled.div`
  font-size: 14px;
  color: #616E85;
  margin-top: 5px;
`;

function Stats({ stats, title }) {
  const formatSize = (bytes) => {
    return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
  };

  const calculateCacheHitRate = () => {
    if (stats.total_requests === 0) return '0%';
    return ((stats.cache_hits / stats.total_requests) * 100).toFixed(1) + '%';
  };

  // Display 304 Not Modified count if available
  const notModifiedCount = stats.not_modified_count !== undefined ? (
    <StatCard>
      <StatValue>{stats.not_modified_count}</StatValue>
      <StatLabel>304 Not Modified</StatLabel>
    </StatCard>
  ) : null;

  return (
    <StatsWrapper>
      {title && <StatsTitle>{title}</StatsTitle>}
      <StatsContainer>
        <StatCard>
          <StatValue>{stats.total_requests}</StatValue>
          <StatLabel>Total Requests</StatLabel>
        </StatCard>
        <StatCard>
          <StatValue>{calculateCacheHitRate()}</StatValue>
          <StatLabel>Cache Hit Rate</StatLabel>
        </StatCard>
        <StatCard>
          <StatValue>{stats.cache_misses}</StatValue>
          <StatLabel>Cache Misses</StatLabel>
        </StatCard>
        <StatCard>
          <StatValue>{formatSize(stats.total_response_size)}</StatValue>
          <StatLabel>Total Data Size</StatLabel>
        </StatCard>
        {notModifiedCount}
      </StatsContainer>
    </StatsWrapper>
  );
}

export default Stats; 