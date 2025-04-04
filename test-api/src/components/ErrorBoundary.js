import React from 'react';
import styled from 'styled-components';

const ErrorContainer = styled.div`
  padding: 20px;
  margin: 20px;
  border: 1px solid #EA3943;
  border-radius: 8px;
  background-color: #FFF2F2;
`;

const ErrorTitle = styled.h2`
  color: #EA3943;
  margin: 0 0 10px 0;
`;

const ErrorDetails = styled.pre`
  background-color: #FFF;
  padding: 10px;
  border-radius: 4px;
  overflow: auto;
  font-size: 14px;
`;

const ReloadButton = styled.button`
  background-color: #EA3943;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
  margin-top: 10px;
  
  &:hover {
    background-color: #D63340;
  }
`;

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({
      error: error,
      errorInfo: errorInfo
    });
    console.error('Error caught by boundary:', error, errorInfo);
  }

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <ErrorContainer>
          <ErrorTitle>Something went wrong</ErrorTitle>
          <p>An error occurred while displaying this content.</p>
          {process.env.NODE_ENV === 'development' && this.state.error && (
            <>
              <ErrorDetails>
                {this.state.error.toString()}
                {this.state.errorInfo.componentStack}
              </ErrorDetails>
            </>
          )}
          <ReloadButton onClick={this.handleReload}>
            Reload Page
          </ReloadButton>
        </ErrorContainer>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary; 