import { FormEvent, useEffect, useState } from 'react';
import type { Channel } from '../types';
import { archiveChannel, createChannel, fetchChannels, joinChannel, leaveChannel } from '../api/client';

interface Props {
  theme: 'light' | 'dark';
  currentUserId?: string;
  onOpenChannel: (channelId: string) => void;
}

export default function ChannelsPanel({ theme, currentUserId, onOpenChannel }: Props) {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [visibility, setVisibility] = useState<'public' | 'private'>('public');
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const dark = theme === 'dark';
  const card = dark ? '#0a2b45' : '#fff';
  const border = dark ? '#275b78' : '#dbe4ea';
  const text = dark ? '#e8eef4' : '#17212b';

  const load = () => fetchChannels().then(setChannels).catch((reason) => setError(reason instanceof Error ? reason.message : 'Failed to load channels'));
  useEffect(load, []);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    setError(null);
    try {
      const channel = await createChannel({ name: name.trim(), description: description.trim(), visibility });
      setChannels((previous) => [...previous, channel]);
      setName('');
      setDescription('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Failed to create channel');
    } finally {
      setCreating(false);
    }
  };

  const toggleMembership = async (channel: Channel) => {
    setError(null);
    try {
      if (channel.joined) await leaveChannel(channel.id); else await joinChannel(channel.id);
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Failed to update membership');
    }
  };

  const archive = async (channel: Channel) => {
    if (!window.confirm(`Archive #${channel.name}? It will no longer appear in channel discovery.`)) return;
    setError(null);
    try {
      await archiveChannel(channel.id);
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Failed to archive channel');
    }
  };

  return (
    <section style={{ flex: 1, color: text, display: 'grid', gap: 20, alignContent: 'start' }}>
      <div>
        <h2 style={{ margin: 0 }}>Channels</h2>
        <p style={{ opacity: .72 }}>Create public or private spaces, manage membership, and open their live conversation.</p>
      </div>
      <form onSubmit={submit} style={{ display: 'grid', gap: 10, padding: 18, border: `1px solid ${border}`, borderRadius: 18, background: card }}>
        <strong>Create a channel</strong>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Channel name" maxLength={100} required style={{ padding: 12, borderRadius: 10, border: `1px solid ${border}` }} />
        <input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What is this channel for?" maxLength={300} style={{ padding: 12, borderRadius: 10, border: `1px solid ${border}` }} />
        <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
          <select value={visibility} onChange={(e) => setVisibility(e.target.value as 'public' | 'private')} style={{ padding: 11, borderRadius: 10 }}>
            <option value="public">Public</option><option value="private">Private</option>
          </select>
          <button type="submit" disabled={creating} style={{ padding: '11px 16px', border: 0, borderRadius: 10, background: '#165c92', color: '#fff', cursor: 'pointer' }}>{creating ? 'Creating…' : 'Create channel'}</button>
        </div>
      </form>
      {error && <div role="alert" style={{ color: '#dc2626' }}>{error}</div>}
      <div style={{ display: 'grid', gap: 12, gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))' }}>
        {channels.map((channel) => (
          <article key={channel.id} style={{ padding: 18, border: `1px solid ${border}`, borderRadius: 18, background: card }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10 }}><strong># {channel.name}</strong><span>{channel.visibility === 'private' ? '🔒' : '🌐'}</span></div>
            <p style={{ minHeight: 38, opacity: .72 }}>{channel.description || 'No description yet.'}</p>
            <div style={{ fontSize: 13, opacity: .7 }}>{channel.memberCount} member{channel.memberCount === 1 ? '' : 's'}</div>
            <div style={{ display: 'flex', gap: 8, marginTop: 14 }}>
              {channel.joined && <button type="button" onClick={() => onOpenChannel(channel.id)} style={{ padding: '9px 12px', border: 0, borderRadius: 9, background: '#165c92', color: '#fff' }}>Open chat</button>}
              <button type="button" disabled={channel.createdBy === currentUserId} onClick={() => toggleMembership(channel)} style={{ padding: '9px 12px', border: `1px solid ${border}`, borderRadius: 9, background: 'transparent', color: text }}>{channel.joined ? 'Leave' : 'Join'}</button>
              {channel.createdBy === currentUserId && <button type="button" onClick={() => archive(channel)} style={{ padding: '9px 12px', border: `1px solid ${border}`, borderRadius: 9, background: 'transparent', color: '#b42318' }}>Archive</button>}
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
