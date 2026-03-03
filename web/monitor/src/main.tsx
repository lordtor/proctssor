import React from 'react';
import ReactDOM from 'react-dom/client';
import Monitor from './Monitor';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <React.StrictMode>
    <Monitor />
  </React.StrictMode>
);
