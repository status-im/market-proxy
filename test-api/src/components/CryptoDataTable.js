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

const DataSource = styled.div`
  padding: 2px 6px;
  border-radius: 4px;
  background-color: ${props => props.$source === 'CoinMarketCap' ? '#3861FB20' : '#16C78420'};
  color: ${props => props.$source === 'CoinMarketCap' ? '#3861FB' : '#16C784'};
  font-size: 10px;
  margin-top: 2px;
`;

function CryptoDataTable({ data, priceData, source }) {
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
            {data.map((item, index) => {
                // Token metadata depends on data source
                const id = item.id;
                const name = item.name;
                const symbol = source === 'CoinMarketCap' ? item.symbol : (item.symbol ? item.symbol.toUpperCase() : '');

                // These fields depend on data source
                let price, volume24h, percentChange24h, marketCap, imageUrl;

                if (source === 'CoinMarketCap') {
                    // CMC format
                    const quote = item?.quote?.USD || {};
                    price = quote.price;
                    volume24h = quote.volume_24h;
                    percentChange24h = quote.percent_change_24h;
                    marketCap = quote.market_cap;
                    imageUrl = `https://s2.coinmarketcap.com/static/img/coins/64x64/${item.id}.png`;
                } else {
                    // CoinGecko format
                    price = item.current_price;
                    volume24h = item.total_volume;
                    percentChange24h = item.price_change_percentage_24h;
                    marketCap = item.market_cap;
                    imageUrl = item.image;
                }

                // Get price data for this token if available
                const tokenPriceData = priceData[symbol];

                return (
                    <tr key={id}>
                        <Td>{index + 1}</Td>
                        <Td>
                            <TokenCell>
                                <TokenImage src={imageUrl} alt={name} />
                                <TokenInfo>
                                    <TokenName>{name}</TokenName>
                                    <TokenSymbol>{symbol}</TokenSymbol>
                                    <DataSource $source={source}>{source}</DataSource>
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

export default CryptoDataTable;