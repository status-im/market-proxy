import React from 'react';
import styled from 'styled-components';

const Container = styled.div`
  max-width: 1200px;
  margin: 0 auto;
  padding: 40px 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
`;

const Header = styled.div`
  text-align: center;
  margin-bottom: 60px;
`;

const Title = styled.h1`
  font-size: 48px;
  font-weight: 600;
  color: #09101c;
  margin: 0 0 16px 0;
  letter-spacing: -0.02em;
`;

const Subtitle = styled.p`
  font-size: 20px;
  color: #647084;
  margin: 0;
  font-weight: 400;
`;

const UtilitiesGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
  margin-top: 40px;
`;

const UtilityCard = styled.div`
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  padding: 32px;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
    border-color: #6366f1;
  }
`;

const UtilityIcon = styled.div`
  width: 48px;
  height: 48px;
  background: ${props => props.color || '#6366f1'};
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
  font-size: 24px;
`;

const UtilityTitle = styled.h3`
  font-size: 20px;
  font-weight: 600;
  color: #09101c;
  margin: 0 0 8px 0;
`;

const UtilityDescription = styled.p`
  font-size: 16px;
  color: #647084;
  line-height: 1.5;
  margin: 0;
`;

const StatusBadge = styled.span`
  display: inline-block;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  margin-top: 12px;
  
  ${props => props.status === 'available' && `
    background: #dcfce7;
    color: #166534;
  `}
  
  ${props => props.status === 'development' && `
    background: #fef3c7;
    color: #92400e;
  `}
`;

const MainPage = ({ onNavigate }) => {
  const utilities = [
    {
      id: 'leaderboard',
      title: 'Leaderboard',
      description: 'Crypto market data dashboard with real-time prices and market statistics from CoinGecko API',
      icon: '📊',
      color: '#6366f1',
      status: 'available'
    },
    {
      id: 'requests-replay',
      title: 'Requests Replay',
      description: 'Tool for replaying HTTP requests from logged NDJSON files to test rate limiting behavior',
      icon: '🔄',
      color: '#8b5cf6',
      status: 'available'
    }
  ];

  return (
    <Container>
      <Header>
        <Title>Market Proxy Utilities</Title>
        <Subtitle>
          Development and testing tools for the Status Market Proxy service
        </Subtitle>
      </Header>

      <UtilitiesGrid>
        {utilities.map((utility) => (
          <UtilityCard
            key={utility.id}
            onClick={() => onNavigate(utility.id)}
          >
            <UtilityIcon color={utility.color}>
              {utility.icon}
            </UtilityIcon>
            <UtilityTitle>{utility.title}</UtilityTitle>
            <UtilityDescription>{utility.description}</UtilityDescription>
            <StatusBadge status={utility.status}>
              {utility.status === 'available' ? 'Available' : 'In Development'}
            </StatusBadge>
          </UtilityCard>
        ))}
      </UtilitiesGrid>
    </Container>
  );
};

export default MainPage;