import { FormEvent, useEffect, useState } from 'react';
import type { Conversation, Task } from '../types';
import { createTask, deleteTask, fetchTasks, updateTaskStatus } from '../api/client';

interface Props { theme: 'light' | 'dark'; conversations: Conversation[]; currentUserId?: string; }

export default function TasksPanel({ theme, conversations, currentUserId }: Props) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [title, setTitle] = useState('');
  const [conversationId, setConversationId] = useState('');
  const [priority, setPriority] = useState('medium');
  const [error, setError] = useState<string | null>(null);
  const dark = theme === 'dark'; const text = dark ? '#e8eef4' : '#17212b'; const border = dark ? '#275b78' : '#dbe4ea'; const card = dark ? '#0a2b45' : '#fff';
  const load = () => fetchTasks().then(setTasks).catch((reason) => setError(reason instanceof Error ? reason.message : 'Failed to load tasks'));
  useEffect(load, []);
  const submit = async (event: FormEvent) => {
    event.preventDefault(); if (!title.trim()) return;
    try {
      const task = await createTask({ title: title.trim(), status: 'todo', priority, conversationId: conversationId || undefined, createdBy: currentUserId ?? '' });
      setTasks((previous) => [task, ...previous]); setTitle('');
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Failed to create task'); }
  };
  const changeStatus = async (task: Task, status: string) => {
    try { const updated = await updateTaskStatus(task.id, status); setTasks((previous) => previous.map((item) => item.id === updated.id ? updated : item)); }
    catch (reason) { setError(reason instanceof Error ? reason.message : 'Failed to update task'); }
  };
  const remove = async (task: Task) => {
    try { await deleteTask(task.id); setTasks((previous) => previous.filter((item) => item.id !== task.id)); }
    catch (reason) { setError(reason instanceof Error ? reason.message : 'Only the creator can delete this task'); }
  };
  return <section style={{ flex: 1, color: text, display: 'grid', gap: 20, alignContent: 'start' }}>
    <div><h2 style={{ margin: 0 }}>Tasks</h2><p style={{ opacity: .72 }}>Assign and track work directly from StatChat conversations.</p></div>
    <form onSubmit={submit} style={{ display: 'grid', gridTemplateColumns: 'minmax(180px,2fr) minmax(150px,1fr) 130px auto', gap: 10, padding: 18, border: `1px solid ${border}`, borderRadius: 18, background: card }}>
      <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Task title" required style={{ padding: 11, borderRadius: 9, border: `1px solid ${border}` }} />
      <select value={conversationId} onChange={(e) => setConversationId(e.target.value)} style={{ padding: 11, borderRadius: 9 }}><option value="">Personal task</option>{conversations.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select>
      <select value={priority} onChange={(e) => setPriority(e.target.value)} style={{ padding: 11, borderRadius: 9 }}><option>low</option><option>medium</option><option>high</option><option>urgent</option></select>
      <button type="submit" style={{ border: 0, borderRadius: 9, background: '#165c92', color: '#fff', padding: '11px 15px' }}>Add task</button>
    </form>
    {error && <div role="alert" style={{ color: '#dc2626' }}>{error}</div>}
    <div style={{ display: 'grid', gap: 10 }}>{tasks.map((task) => <article key={task.id} style={{ display: 'grid', gridTemplateColumns: '1fr auto auto', alignItems: 'center', gap: 12, padding: 16, border: `1px solid ${border}`, borderRadius: 14, background: card }}>
      <div><strong>{task.title}</strong><div style={{ opacity: .65, fontSize: 13 }}>{task.priority} priority{task.dueDate ? ` · due ${task.dueDate}` : ''}</div></div>
      <select value={task.status} onChange={(e) => changeStatus(task, e.target.value)}><option value="todo">To do</option><option value="in_progress">In progress</option><option value="blocked">Blocked</option><option value="done">Done</option></select>
      <button type="button" onClick={() => remove(task)} disabled={task.createdBy !== currentUserId} aria-label={`Delete ${task.title}`} style={{ border: 0, background: 'transparent', color: '#dc2626' }}>Delete</button>
    </article>)}</div>
  </section>;
}
