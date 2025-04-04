import React from 'react';
import styled from 'styled-components';

const TabsContainer = styled.div`
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
  border-bottom: 1px solid #eee;
  padding-bottom: 10px;
`;

const Tab = styled.div`
  cursor: pointer;
  color: ${props => props.$active ? '#3861FB' : '#616E85'};
  font-weight: ${props => props.$active ? 'bold' : 'normal'};
  
  &:hover {
    color: #3861FB;
  }
`;

function Tabs({ tabs, activeTab, onTabChange }) {
    return (
        <TabsContainer>
            {tabs.map(tab => (
                <Tab
                    key={tab}
                    $active={activeTab === tab}
                    onClick={() => onTabChange(tab)}
                >
                    {tab}
                </Tab>
            ))}
        </TabsContainer>
    );
}

export default Tabs; 