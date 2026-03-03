import React from 'react';

interface LoadingSpinnerProps {
  fullScreen?: boolean;
  size?: 'small' | 'medium' | 'large';
  message?: string;
}

const sizes = {
  small: 20,
  medium: 40,
  large: 60,
};

const styles = {
  container: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    justifyContent: 'center',
    gap: '15px',
  },
  fullScreen: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(255, 255, 255, 0.9)',
    zIndex: 9999,
  },
  spinner: {
    width: '60px',
    height: '60px',
    border: '4px solid #f3f3f3',
    borderTop: '4px solid #4ecca3',
    borderRadius: '50%',
    animation: 'spin 1s linear infinite',
  },
  message: {
    color: '#666',
    fontSize: '14px',
  },
};

// Add keyframe animation via style tag
const styleSheet = document.createElement('style');
styleSheet.textContent = `
  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }
`;
if (typeof document !== 'undefined') {
  document.head.appendChild(styleSheet);
}

export default function LoadingSpinner({ 
  fullScreen = false, 
  size = 'medium',
  message 
}: LoadingSpinnerProps) {
  const spinnerSize = sizes[size];

  return (
    <div 
      style={{
        ...styles.container,
        ...(fullScreen ? styles.fullScreen : {}),
        width: fullScreen ? '100vw' : '100%',
        height: fullScreen ? '100vh' : '100%',
      }}
    >
      <div 
        style={{
          ...styles.spinner,
          width: spinnerSize,
          height: spinnerSize,
        }} 
      />
      {message && <div style={styles.message}>{message}</div>}
    </div>
  );
}
