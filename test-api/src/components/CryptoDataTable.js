import React, { useState } from 'react';
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

const PriceSource = styled.div`
  padding: 1px 4px;
  border-radius: 3px;
  background-color: ${props => props.$endpoint === 'prices' ? '#3861FB15' : '#16C78415'};
  color: ${props => props.$endpoint === 'prices' ? '#3861FB' : '#16C784'};
  font-size: 9px;
  margin-top: 1px;
  display: inline-block;
`;

const DataSource = styled.div`
  padding: 2px 6px;
  border-radius: 4px;
  background-color: ${props => props.$source === 'CoinMarketCap' ? '#3861FB20' : '#16C78420'};
  color: ${props => props.$source === 'CoinMarketCap' ? '#3861FB' : '#16C784'};
  font-size: 10px;
  margin-top: 2px;
`;

// Pagination components
const PaginationContainer = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
`;

const PageButtons = styled.div`
  display: flex;
  gap: 5px;
`;

const PageButton = styled.button`
  padding: 8px 12px;
  border: 1px solid #eee;
  background-color: ${props => props.$active ? '#3861FB' : 'white'};
  color: ${props => props.$active ? 'white' : '#333'};
  border-radius: 4px;
  cursor: pointer;
  
  &:hover {
    background-color: ${props => props.$active ? '#3861FB' : '#f5f5f5'};
  }
  
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

const PageSizeSelector = styled.select`
  padding: 8px 12px;
  border: 1px solid #eee;
  border-radius: 4px;
  background-color: white;
`;

const PageInfo = styled.div`
  color: #616E85;
`;

function CryptoDataTable({ data, priceData, source, priceEndpoint }) {
    const [currentPage, setCurrentPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);

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
    
    // Calculate pagination values
    const totalItems = data.length;
    const totalPages = Math.ceil(totalItems / pageSize);
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = Math.min(startIndex + pageSize, totalItems);
    const currentData = data.slice(startIndex, endIndex);
    
    // Handle page change
    const handlePageChange = (page) => {
        setCurrentPage(page);
    };
    
    // Handle page size change
    const handlePageSizeChange = (e) => {
        const newPageSize = parseInt(e.target.value);
        setPageSize(newPageSize);
        setCurrentPage(1); // Reset to first page when changing page size
    };
    
    // Generate page numbers
    const getPageNumbers = () => {
        const pages = [];
        const maxVisiblePages = 5;
        
        // Always show first page
        pages.push(1);
        
        // Calculate range of visible pages
        let startPage = Math.max(2, currentPage - Math.floor(maxVisiblePages / 2));
        let endPage = Math.min(totalPages - 1, startPage + maxVisiblePages - 3);
        
        // Adjust if at the beginning or end
        if (startPage > 2) {
            pages.push('...');
        }
        
        // Add middle pages
        for (let i = startPage; i <= endPage; i++) {
            pages.push(i);
        }
        
        // Add end ellipsis if needed
        if (endPage < totalPages - 1) {
            pages.push('...');
        }
        
        // Always show last page if more than 1 page
        if (totalPages > 1) {
            pages.push(totalPages);
        }
        
        return pages;
    };

    return (
        <>
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
                {currentData.map((item, index) => {
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
                    // Try both symbol (for 'prices' endpoint) and id (for 'simpleprices' endpoint)
                    const tokenPriceData = priceData[symbol] || priceData[id];

                    return (
                        <tr key={id}>
                            <Td>{startIndex + index + 1}</Td>
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
                                        <SecondaryPrice>
                                            {formatNumber(tokenPriceData.price)}
                                            <PriceSource $endpoint={priceEndpoint}>{priceEndpoint}</PriceSource>
                                        </SecondaryPrice>
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
                                            <PriceSource $endpoint={priceEndpoint}>{priceEndpoint}</PriceSource>
                                        </SecondaryPrice>
                                    )}
                                </PriceContainer>
                            </Td>
                            <Td>
                                <PriceContainer>
                                    <MainPrice>{formatNumber(volume24h)}</MainPrice>
                                    {tokenPriceData?.volume_24h && (
                                        <SecondaryPrice>
                                            {formatNumber(tokenPriceData.volume_24h)}
                                            <PriceSource $endpoint={priceEndpoint}>{priceEndpoint}</PriceSource>
                                        </SecondaryPrice>
                                    )}
                                </PriceContainer>
                            </Td>
                            <Td>
                                <PriceContainer>
                                    <MainPrice>{formatNumber(marketCap)}</MainPrice>
                                    {tokenPriceData?.market_cap && (
                                        <SecondaryPrice>
                                            {formatNumber(tokenPriceData.market_cap)}
                                            <PriceSource $endpoint={priceEndpoint}>{priceEndpoint}</PriceSource>
                                        </SecondaryPrice>
                                    )}
                                </PriceContainer>
                            </Td>
                            <Td>
                                <SwapButton>Swap</SwapButton>
                            </Td>
                        </tr>
                    );
                })}
                </tbody>
            </Table>
            
            {/* Pagination */}
            <PaginationContainer>
                <PageInfo>
                    Showing {startIndex + 1}-{endIndex} of {totalItems} items
                </PageInfo>
                
                <PageButtons>
                    <PageButton 
                        onClick={() => handlePageChange(1)} 
                        disabled={currentPage === 1}
                    >
                        «
                    </PageButton>
                    <PageButton 
                        onClick={() => handlePageChange(currentPage - 1)} 
                        disabled={currentPage === 1}
                    >
                        ‹
                    </PageButton>
                    
                    {getPageNumbers().map((page, index) => (
                        page === '...' 
                            ? <PageButton key={`ellipsis-${index}`} disabled>...</PageButton>
                            : <PageButton 
                                key={page} 
                                $active={page === currentPage}
                                onClick={() => handlePageChange(page)}
                              >
                                {page}
                              </PageButton>
                    ))}
                    
                    <PageButton 
                        onClick={() => handlePageChange(currentPage + 1)} 
                        disabled={currentPage === totalPages}
                    >
                        ›
                    </PageButton>
                    <PageButton 
                        onClick={() => handlePageChange(totalPages)} 
                        disabled={currentPage === totalPages}
                    >
                        »
                    </PageButton>
                </PageButtons>
                
                <PageSizeSelector value={pageSize} onChange={handlePageSizeChange}>
                    <option value="10">10 per page</option>
                    <option value="25">25 per page</option>
                    <option value="50">50 per page</option>
                    <option value="100">100 per page</option>
                </PageSizeSelector>
            </PaginationContainer>
        </>
    );
}

export default CryptoDataTable;