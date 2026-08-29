import { useEffect, useState } from 'react';
import { fetchTranslationLanguages, translateText, type TranslationLanguage, type TranslationResult } from '../api/client';
import styles from './TranslationPanel.module.css';

interface Props {
  theme: 'light' | 'dark';
}

export default function TranslationPanel({ theme }: Props) {
  const [languages, setLanguages] = useState<TranslationLanguage[]>([]);
  const [sourceLanguage, setSourceLanguage] = useState('en');
  const [targetLanguage, setTargetLanguage] = useState('sw');
  const [text, setText] = useState('Hello team, the meeting is today.');
  const [result, setResult] = useState<TranslationResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const isDark = theme === 'dark';

  useEffect(() => {
    fetchTranslationLanguages().then(setLanguages).catch(() => setError('Translation languages could not be loaded'));
  }, []);

  const translate = async () => {
    if (!text.trim() || sourceLanguage === targetLanguage) return;
    setBusy(true);
    setError('');
    try {
      setResult(await translateText({ text: text.trim(), sourceLanguage, targetLanguage }));
    } catch {
      setError('Translation could not be completed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className={styles.shell} style={{ background: isDark ? '#0a2b45' : '#fff', color: isDark ? '#e8eef4' : '#17212b' }}>
      <div className={styles.header}>
        <div><h2>Translation</h2><p>Translate short collaboration notes without leaving StatChat.</p></div>
        <span className={styles.provider}>Local glossary ready</span>
      </div>
      <div className={styles.controls}>
        <label>From<select value={sourceLanguage} onChange={(event) => setSourceLanguage(event.target.value)}>{languages.map((language) => <option key={language.code} value={language.code}>{language.name}</option>)}</select></label>
        <button type="button" className={styles.swap} onClick={() => { setSourceLanguage(targetLanguage); setTargetLanguage(sourceLanguage); setResult(null); }}>Swap</button>
        <label>To<select value={targetLanguage} onChange={(event) => setTargetLanguage(event.target.value)}>{languages.map((language) => <option key={language.code} value={language.code}>{language.name}</option>)}</select></label>
      </div>
      <textarea className={styles.input} value={text} maxLength={10000} onChange={(event) => setText(event.target.value)} placeholder="Write a message or note to translate" />
      <button type="button" className={styles.translateButton} onClick={translate} disabled={busy || !text.trim() || sourceLanguage === targetLanguage}>{busy ? 'Translating...' : 'Translate'}</button>
      {result && <div className={styles.result} aria-live="polite"><span>{result.sourceLanguage.toUpperCase()} to {result.targetLanguage.toUpperCase()} · {result.provider}</span><p>{result.text}</p></div>}
      {error && <p className={styles.error} role="alert">{error}</p>}
    </section>
  );
}
