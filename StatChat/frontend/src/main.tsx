import React from 'react';
import ReactDOM from 'react-dom/client';
import './global.css';
import App from './App';
import Login from './components/Login';
import { bootstrapSharedSignOn, getAuthToken } from './api/client';

bootstrapSharedSignOn();

const hasToken = () => !!getAuthToken();

function Root() {
  // Re-render when a token appears/disappears (login/logout).
  const [authed, setAuthed] = React.useState(hasToken);
  React.useEffect(() => {
    const onStorage = () => setAuthed(hasToken());
    window.addEventListener('storage', onStorage);
    window.addEventListener('statchat-auth', onStorage);
    return () => {
      window.removeEventListener('storage', onStorage);
      window.removeEventListener('statchat-auth', onStorage);
    };
  }, []);
  return authed ? <App /> : <Login />;
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Root />
  </React.StrictMode>
);
