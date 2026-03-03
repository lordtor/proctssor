import React from 'react';

interface User {
  id: string;
  name: string;
  email: string;
  avatar?: string;
}

interface HeaderProps {
  user?: User | null;
}

const styles = {
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '15px 25px',
    backgroundColor: '#fff',
    borderBottom: '1px solid #e0e0e0',
    boxShadow: '0 2px 4px rgba(0,0,0,0.05)',
  },
  left: {
    display: 'flex',
    alignItems: 'center',
    gap: '20px',
  },
  logo: {
    fontSize: '20px',
    fontWeight: 700,
    color: '#1a1a2e',
  },
  breadcrumb: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '14px',
    color: '#666',
  },
  breadcrumbSeparator: {
    color: '#999',
  },
  right: {
    display: 'flex',
    alignItems: 'center',
    gap: '15px',
  },
  iconButton: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: '36px',
    height: '36px',
    border: 'none',
    borderRadius: '8px',
    backgroundColor: '#f5f5f5',
    cursor: 'pointer',
    fontSize: '16px',
    transition: 'all 0.2s',
  },
  userInfo: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    padding: '6px 12px',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  avatar: {
    width: '32px',
    height: '32px',
    borderRadius: '50%',
    backgroundColor: '#4ecca3',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#1a1a2e',
    fontWeight: 600,
    fontSize: '14px',
  },
  userName: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
  },
  userEmail: {
    fontSize: '12px',
    color: '#666',
  },
};

export default function Header({ user }: HeaderProps) {
  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const handleLogout = () => {
    localStorage.removeItem('auth_token');
    window.location.href = '/login';
  };

  return (
    <header style={styles.header}>
      <div style={styles.left}>
        <div style={styles.logo}>⚡ Workflow</div>
      </div>

      <div style={styles.right}>
        <button style={styles.iconButton} title="Notifications">
          🔔
        </button>
        <button style={styles.iconButton} title="Settings">
          ⚙️
        </button>
        
        {user ? (
          <div style={styles.userInfo} onClick={handleLogout} title="Click to logout">
            <div style={styles.avatar}>
              {user.avatar ? (
                <img src={user.avatar} alt={user.name} style={{ width: '100%', height: '100%', borderRadius: '50%' }} />
              ) : (
                getInitials(user.name)
              )}
            </div>
            <div>
              <div style={styles.userName}>{user.name}</div>
              <div style={styles.userEmail}>{user.email}</div>
            </div>
          </div>
        ) : (
          <button
            style={{ ...styles.iconButton, width: 'auto', padding: '0 15px', fontSize: '14px' }}
            onClick={() => window.location.href = '/login'}
          >
            Login
          </button>
        )}
      </div>
    </header>
  );
}
