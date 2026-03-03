import { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Navigation from './components/Navigation';

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
            <Routes>
              <Route path="/" element={<div>Welcome to Workflow Platform</div>} />
              <Route path="/modeler/*" element={<Modeler />} />
              <Route path="/tasklist/*" element={<Tasklist />} />
              <Route path="/monitor/*" element={<Monitor />} />
            </Routes>
          </Suspense>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
