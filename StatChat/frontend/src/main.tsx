import React from 'react';
import ReactDOM from 'react-dom/client';
import './global.css';
import App from './App';
import { bootstrapSharedSignOn } from './api/client';

bootstrapSharedSignOn();

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
