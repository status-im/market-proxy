import React from 'react';
import styled from 'styled-components';

const Container = styled.div`
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  min-height: 100vh;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
`;

function Layout({ children, title }) {
    return (
        <Container>
            {children}
        </Container>
    );
}

export default Layout; 