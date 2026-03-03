import React, { useState, useEffect } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { api } from '../../../shared/src/lib/api';

interface ServiceInfo {
  name: string;
  version: string;
}

interface SidebarProps {
  services?: ServiceInfo[];
}

const styles = {
  nav: {
    width: '260px',
    backgroundColor: '#1a1a2e',
    minHeight: '100vh',
    padding: '20px',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  title: {
    color: '#fff',
    fontSize: '22px',
    fontWeight: 'bold',
    marginBottom: '30px',
    padding: '0 10px',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  sectionTitle: {
    color: '#666',
    fontSize: '11px',
    textTransform: 'uppercase' as const,
    marginTop: '20px',
    marginBottom: '10px',
    paddingLeft: '15px',
    letterSpacing: '0.5px',
  },
  link: {
    color: '#a0a0a0',
    textDecoration: 'none',
    padding: '12px 15px',
    borderRadius: '8px',
    marginBottom: '6px',
    fontSize: '14px',
    transition: 'all 0.2s',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  activeLink: {
    backgroundColor: '#16213e',
    color: '#4ecca3',
  },
  servicesSection: {
    marginTop: 'auto',
    borderTop: '1px solid #2a2a4a',
    paddingTop: '15px',
  },
  serviceItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    padding: '8px 15px',
    color: '#888',
    fontSize: '12px',
    borderRadius: '6px',
    marginBottom: '4px',
  },
  serviceDot: {
    width: '6px',
    height: '6px',
    borderRadius: '50%',
    backgroundColor: '#4ecca3',
  },
  footer: {
    padding: '15px',
    borderTop: '1px solid #2a2a4a',
    marginTop: '15px',
  },
  version: {
    fontSize: '11px',
    color: '#555',
    textAlign: 'center' as const,
  },
};

export default function Sidebar({ services = [] }: SidebarProps) {
  const location = useLocation();
  const [currentServices, setCurrentServices] = useState<ServiceInfo[]>(services);

  // Load services from registry if not provided
  useEffect(() => {
    if (services.length > 0) return;
    
    api.registry.listServices()
      .then(setCurrentServices)
      .catch(err => console.warn('Failed to load services:', err));
  }, [services]);

  return (
    <nav style={styles.nav}>
      <div style={styles.title}>
        <span>⚡</span>
        <span>Workflow</span>
      </div>
      
      <div style={styles.sectionTitle}>Applications</div>
      
      <NavLink 
        to="/" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
        end
      >
        🏠 <span>Home</span>
      </NavLink>
      
      <NavLink 
        to="/modeler" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
      >
        ✏️ <span>Modeler</span>
      </NavLink>
      
      <NavLink 
        to="/tasklist" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
      >
        📋 <span>Tasklist</span>
      </NavLink>
      
      <NavLink 
        to="/monitor" 
        style={({ isActive }) => ({
          ...styles.link,
          ...(isActive ? styles.activeLink : {}),
        })}
      >
        📊 <span>Monitor</span>
      </NavLink>

      {currentServices.length > 0 && (
        <div style={styles.servicesSection}>
          <div style={styles.sectionTitle}>Registered Services</div>
          {currentServices.map((service) => (
            <div key={service.name} style={styles.serviceItem}>
              <div style={styles.serviceDot} />
              <span>{service.name}</span>
              <span style={{ color: '#555', marginLeft: 'auto' }}>v{service.version}</span>
            </div>
          ))}
        </div>
      )}

      <div style={styles.footer}>
        <div style={styles.version}>Workflow Platform v1.0.0</div>
      </div>
    </nav>
  );
}
