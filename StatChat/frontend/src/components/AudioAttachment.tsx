import { useEffect, useRef, useState } from 'react';
import type { MessageAttachment } from '../types';
import styles from './ChatWorkspace.module.css';

interface Props {
  attachment: MessageAttachment;
}

function formatTime(seconds: number) {
  const rounded = Math.floor(seconds);
  const mins = Math.floor(rounded / 60);
  const secs = rounded % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

export default function AudioAttachment({ attachment }: Props) {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    const handleTimeUpdate = () => setCurrentTime(audio.currentTime);
    const handleLoadedMetadata = () => setDuration(audio.duration);
    const handleEnded = () => setPlaying(false);

    audio.addEventListener('timeupdate', handleTimeUpdate);
    audio.addEventListener('loadedmetadata', handleLoadedMetadata);
    audio.addEventListener('ended', handleEnded);

    return () => {
      audio.removeEventListener('timeupdate', handleTimeUpdate);
      audio.removeEventListener('loadedmetadata', handleLoadedMetadata);
      audio.removeEventListener('ended', handleEnded);
    };
  }, [attachment.url]);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;
    if (playing) {
      const playPromise = audio.play();
      if (playPromise?.catch) {
        playPromise.catch(() => {
          setPlaying(false);
        });
      }
    } else {
      audio.pause();
    }
  }, [playing]);

  const togglePlay = () => {
    setPlaying((prev) => !prev);
  };

  const isVoiceNote = attachment.fileName.toLowerCase().startsWith('voice-note');
  const label = isVoiceNote ? 'Voice note' : attachment.fileName;

  return (
    <div className={styles.audioAttachment}>
      <div className={styles.audioAttachmentHeader}>
        <button
          type="button"
          className={styles.audioPlayButton}
          onClick={togglePlay}
          aria-label={playing ? 'Pause audio' : 'Play audio'}
        >
          {playing ? '❚❚' : '▶'}
        </button>
        <div>
          <div className={styles.audioAttachmentTitle}>{label}</div>
          <div className={styles.audioAttachmentMeta}>
            {formatTime(currentTime)} / {formatTime(duration || 0)}
          </div>
        </div>
      </div>
      <progress
        className={styles.audioProgress}
        max={duration || 1}
        value={Math.min(currentTime, duration || 0)}
      />
      <audio ref={audioRef} src={attachment.url} preload="metadata" className={styles.hiddenAudio} />
    </div>
  );
}
