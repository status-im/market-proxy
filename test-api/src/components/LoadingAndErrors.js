import React from 'react';
import styled from 'styled-components';

const LoadingMessage = styled.div`
  text-align: center;
  padding: 20px;
  color: #616E85;
`;

const ErrorMessage = styled.div`
  text-align: center;
  padding: 20px;
  color: #EA3943;
`;

export const Loading = () => (
  <LoadingMessage>Loading cryptocurrency data...</LoadingMessage>
);

export const Error = ({ message }) => (
  <ErrorMessage>{message}</ErrorMessage>
); 