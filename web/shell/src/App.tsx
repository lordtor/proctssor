import { Suspense, lazy, Component, ReactNode } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Navigation from './components/Navigation';

// ErrorBoundary component for handling lazy loading errors
class ErrorBoundary extends Component<{ children: ReactNode; fallback: ReactNode }, { hasError: boolean }> {
  constructor(props: { children: ReactNode; fallback: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }
    return this.props.children;
  }
}

const Modeler = lazy(() => import('@wf/modeler/Modeler'));
const Tasklist = lazy(() => import('@wf/tasklist/Tasklist'));
const Monitor = lazy(() => import('@wf/monitor/Monitor'));

function Loading() {
  return (
    <div style={{ 
      display: 'flex', 
      justifyContent: 'center', 
      alignItems: 'center', 
      height: '100vh' 
    }}>
      Loading...
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <div style={{ display: 'flex', minHeight: '100vh' }}>
        <Navigation />
        <main style={{ flex: 1, padding: '20px' }}>
          <Suspense fallback={<Loading />}>
            <ErrorBoundary
              fallback={
                <div style={{ 
                  padding: '20px', 
                  color: '#721c24', 
                  backgroundColor: '#f8d7da',
                  borderRadius: '8px',
                  textAlign: 'center'
                }}>
                  Failed to load this module. Please refresh the page or try again later.
                </div>
              }
            >
              <Routes>
                <Route path="/" element={<div>Welcome to Workflow Platform</div>} />
                <Route path="/modeler/*" element={<Modeler />} />
                <Route path="/tasklist/*" element={<Tasklist />} />
                <Route path="/monitor/*" element={<Monitor />} />
              </Routes>
            </ErrorBoundary>
          </Suspense>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
