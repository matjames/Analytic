import styles from './GoogleMeetProvider.module.css';

interface Props {
  connected: boolean;
  onConnect: () => void;
  theme: 'light' | 'dark';
}

export default function GoogleMeetProvider({ connected, onConnect, theme }: Props) {
  const isDark = theme === 'dark';
  return (
    <div className={styles.providerCard} style={{ background: isDark ? '#0f3f5f' : '#ffffff', borderColor: isDark ? '#6b7280' : '#e5e7eb' }}>
      <div className={styles.providerIcon}>🎥</div>
      <div className={styles.providerName}>Google Meet</div>
      <div
        className={styles.providerStatus}
        style={{ background: connected ? '#d1fae5' : '#fef3c7', color: connected ? '#166534' : '#92400e' }}
      >
        {connected ? 'Connected' : 'Connect to use'}
      </div>
      <button
        type="button"
        className={styles.connectBtn}
        onClick={onConnect}
        title={connected ? 'Disconnect Google Meet' : 'Connect Google Meet'}
      >
        {connected ? 'Disconnect' : 'Connect Google Meet'}
      </button>
    </div>
  );
}