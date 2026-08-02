import styles from './ZoomProvider.module.css';

interface Props {
  connected: boolean;
  onConnect: () => void;
  theme: 'light' | 'dark';
}

export default function ZoomProvider({ connected, onConnect, theme }: Props) {
  const isDark = theme === 'dark';
  return (
    <div className={styles.providerCard} style={{ background: isDark ? '#0f3f5f' : '#ffffff', borderColor: isDark ? '#6b7280' : '#e5e7eb' }}>
      <div className={styles.providerIcon}>📹</div>
      <div className={styles.providerName}>Zoom</div>
      <div className={styles.providerStatus} style={{ background: connected ? '#dcfce7' : '#fee2e2', color: connected ? '#166534' : '#991b1b' }}>
        {connected ? '✓ Connected' : 'Not Connected'}
      </div>
      {!connected && (
        <button type="button" className={styles.connectBtn} onClick={onConnect}>
          Connect Zoom
        </button>
      )}
    </div>
  );
}