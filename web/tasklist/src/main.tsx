import React from 'react';
import ReactDOM from 'react-dom/client';
import Tasklist from './Tasklist';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <React.StrictMode>
    <Tasklist />
  </React.StrictMode>
);
