import React from 'react';
import styled from 'styled-components';

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
`;

const Th = styled.th`
  text-align: left;
  padding: 15px;
  color: #616E85;
  font-weight: normal;
`;

const Td = styled.td`
  padding: 15px;
  border-top: 1px solid #eee;
`;

const TokenCell = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
`;

const TokenImage = styled.img`
  width: 24px;
  height: 24px;
  border-radius: 50%;
`;

const TokenInfo = styled.div`
  display: flex;
  flex-direction: column;
`;

const TokenName = styled.span`
  font-weight: 500;
`;

const TokenSymbol = styled.span`
  color: #616E85;
  font-size: 0.9em;
`;

const PercentageChange = styled.span`
  color: ${props => props.$value >= 0 ? '#16C784' : '#EA3943'};
`;

const SwapButton = styled.button`
  background-color: #EEF2FE;
  color: #3861FB;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 500;
  
  &:hover {
    background-color: #D8E1FF;
  }
`;

const PriceContainer = styled.div`
  display: flex;
  flex-direction: column;
`;

const MainPrice = styled.div`
  font-weight: 500;
`;

const SecondaryPrice = styled.div`
  color: #9295A6;
  font-size: 0.85em;
  margin-top: 2px;
`;

function CryptoTable({ cryptoData, priceData }) {
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
    return `${value >= 0 ? '↑' : '↓'} ${Math.abs(value).toFixed(2)}%`;
  };

  return (
      <Table>
        <thead>
        <tr>
          <Th>#</Th>
          <Th>Token</Th>
          <Th>Price</Th>
          <Th>24hr</Th>
          <Th>24hr Volume</Th>
          <Th>Market Cap</Th>
          <Th></Th>
        </tr>
        </thead>
        <tbody>
        {cryptoData.map((crypto, index) => {
          // Safely access nested properties from main crypto data
          const quote = crypto?.quote?.USD || {};
          const price = quote.price;
          const volume24h = quote.volume_24h;
          const marketCap = quote.market_cap;
          const percentChange24h = quote.percent_change_24h;

          // Get price data for this token if available
          const tokenPriceData = priceData[crypto.symbol];

          return (
              <tr key={crypto.id}>
                <Td>{index + 1}</Td>
                <Td>
                  <TokenCell>
                    <TokenImage src={`https://s2.coinmarketcap.com/static/img/coins/64x64/${crypto.id}.png`} alt={crypto.name} />
                    <TokenInfo>
                      <TokenName>{crypto.name}</TokenName>
                      <TokenSymbol>{crypto.symbol}</TokenSymbol>
                    </TokenInfo>
                  </TokenCell>
                </Td>
                <Td>
                  <PriceContainer>
                    <MainPrice>{formatNumber(price)}</MainPrice>
                    {tokenPriceData?.price && (
                        <SecondaryPrice>{formatNumber(tokenPriceData.price)}</SecondaryPrice>
                    )}
                  </PriceContainer>
                </Td>
                <Td>
                  <PriceContainer>
                    <PercentageChange $value={percentChange24h}>
                      {percentChange24h ? formatPercentage(percentChange24h) : '—'}
                    </PercentageChange>
                    {tokenPriceData?.percent_change_24h !== undefined && (
                        <SecondaryPrice>
                          {formatPercentage(tokenPriceData.percent_change_24h)}
                        </SecondaryPrice>
                    )}
                  </PriceContainer>
                </Td>
                <Td>
                  <PriceContainer>
                    <MainPrice>{formatNumber(volume24h)}</MainPrice>
                    {tokenPriceData?.volume_24h !== undefined && (
                        <SecondaryPrice>{formatNumber(tokenPriceData.volume_24h)}</SecondaryPrice>
                    )}
                  </PriceContainer>
                </Td>
                <Td>{formatNumber(marketCap)}</Td>
                <Td>
                  <SwapButton>Swap</SwapButton>
                </Td>
              </tr>
          );
        })}
        </tbody>
      </Table>
  );
}

export default CryptoTable; 