import React, { ButtonHTMLAttributes } from 'react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}

const variants = {
  primary: {
    backgroundColor: '#4ecca3',
    color: '#1a1a2e',
    border: 'none',
  },
  secondary: {
    backgroundColor: '#1a1a2e',
    color: '#fff',
    border: 'none',
  },
  danger: {
    backgroundColor: '#dc3545',
    color: '#fff',
    border: 'none',
  },
  ghost: {
    backgroundColor: 'transparent',
    color: '#1a1a2e',
    border: 'none',
  },
  outline: {
    backgroundColor: 'transparent',
    color: '#1a1a2e',
    border: '1px solid #ddd',
  },
};

const sizes = {
  sm: { padding: '6px 12px', fontSize: '12px' },
  md: { padding: '10px 16px', fontSize: '14px' },
  lg: { padding: '14px 24px', fontSize: '16px' },
};

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  leftIcon,
  rightIcon,
  children,
  disabled,
  style,
  ...props
}: ButtonProps) {
  return (
    <button
      disabled={disabled || loading}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '8px',
        borderRadius: '8px',
        fontWeight: 600,
        cursor: disabled || loading ? 'not-allowed' : 'pointer',
        opacity: disabled || loading ? 0.6 : 1,
        transition: 'all 0.2s',
        ...variants[variant],
        ...sizes[size],
        ...style,
      }}
      {...props}
    >
      {loading && (
        <span style={{ animation: 'spin 1s linear infinite' }}>⏳</span>
      )}
      {leftIcon && <span>{leftIcon}</span>}
      <span>{children}</span>
      {rightIcon && <span>{rightIcon}</span>}
      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </button>
  );
}

export default Button;
