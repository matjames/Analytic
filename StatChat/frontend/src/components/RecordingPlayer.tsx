interface Props {
  url: string;
  onClose: () => void;
}

export default function RecordingPlayer({ url, onClose }: Props) {
  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 150,
        background: 'rgba(0,0,0,0.8)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: '#0a0f1a',
          borderRadius: 16,
          padding: 16,
          maxWidth: 860,
          width: '100%',
          border: '1px solid rgba(255,255,255,0.15)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <strong style={{ color: '#fff', fontSize: 16 }}>Recording Playback</strong>
          <button
            type="button"
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              color: '#fff',
              fontSize: 18,
              cursor: 'pointer',
            }}
            title="Close"
          >
            ✕
          </button>
        </div>
        <video
          src={url}
          controls
          autoPlay
          style={{ width: '100%', maxHeight: '70vh', borderRadius: 12, background: '#000' }}
        />
      </div>
    </div>
  );
}
