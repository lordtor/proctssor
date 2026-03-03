import { NavLink } from 'react-router-dom';

const styles = {
  nav: {
    width: '250px',
    backgroundColor: '#1a1a2e',
    minHeight: '100vh',
    padding: '20px',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  title: {
    color: '#fff',
    fontSize: '24px',
    fontWeight: 'bold',
    marginBottom: '30px',
    padding: '0 10px',
  },
  link: {
    color: '#a0a0a0',
    textDecoration: 'none',
    padding: '12px 15px',
    borderRadius: '8px',
    marginBottom: '8px',
    fontSize: '16px',
    transition: 'all 0.2s',
  },
  activeLink: {
    backgroundColor: '#16213e',
    color: '#4ecca3',
  },
  sectionTitle: {
    color: '#666',
    fontSize: '12px',
    textTransform: 'uppercase' as const,
    marginTop: '20px',
    marginBottom: '10px',
    paddingLeft: '15px',
  },
};

function Navigation() {
  return (
    <nav style={styles.nav}>
      <div style={styles.title}>Workflow</div>
      
      <div style={styles.sectionTitle}>Applications</div>
      
      <NavLink 
        to="/" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
        end
      >
        Home
      </NavLink>
      
      <NavLink 
        to="/modeler" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
      >
        Modeler
      </NavLink>
      
      <NavLink 
        to="/tasklist" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
      >
        Tasklist
      </NavLink>
      
      <NavLink 
        to="/monitor" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
      >
        Monitor
      </NavLink>
    </nav>
  );
}

export default Navigation;
