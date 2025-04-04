import React from 'react';
import styled from 'styled-components';

const Container = styled.div`
  padding: 20px;
  background-color: #fff;
  min-height: 100vh;
`;

const Header = styled.h1`
  font-size: 24px;
  margin-bottom: 20px;
`;

function Layout({ children, title }) {
    return (
        <Container>
            <Header>{title}</Header>
            {children}
        </Container>
    );
}

export default Layout; 