import React, { useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

export default function KanbanBoard({ tasks, onTaskUpdate, projectId }) {
  const [isAdding, setIsAdding] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [newPriority, setNewPriority] = useState('Medium');
  const [newAssignee, setNewAssignee] = useState('');
  const [newWBS, setNewWBS] = useState('1.1');

  const columns = ['Todo', 'In Progress', 'Review', 'Done'];

  const getTasksByStatus = (status) => {
    return tasks.filter(t => t.status === status);
  };

  const handleMove = (task, targetStatus) => {
    let progress = 0;
    if (targetStatus === 'In Progress') progress = 25;
    if (targetStatus === 'Review') progress = 80;
    if (targetStatus === 'Done') progress = 100;

    fetch(`${API_BASE}/api/tasks/${task.id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: targetStatus, progress })
    })
      .then(res => res.json())
      .then(data => {
        onTaskUpdate();
      })
      .catch(err => console.error("Error updating task status:", err));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!newTitle.trim()) return;

    const payload = {
      projectId,
      wbsCode: newWBS,
      title: newTitle,
      description: newDesc,
      status: 'Todo',
      priority: newPriority,
      startDate: new Date().toISOString().split('T')[0],
      endDate: new Date(Date.now() + 7*24*60*60*1000).toISOString().split('T')[0],
      progress: 0,
      assignedTo: newAssignee
    };

    fetch(`${API_BASE}/api/tasks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
      .then(res => res.json())
      .then(() => {
        onTaskUpdate();
        setIsAdding(false);
        setNewTitle('');
        setNewDesc('');
        setNewAssignee('');
      })
      .catch(err => console.error("Error creating task:", err));
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 style={{ fontSize: '18px', fontWeight: '700' }}>Task Kanban Board</h3>
        <button className="btn btn-primary" onClick={() => setIsAdding(true)}>+ Add Task</button>
      </div>

      {isAdding && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleSubmit}>
            <h3 style={{ marginBottom: '10px' }}>Create New Task / Component Activity</h3>
            
            <div className="form-row">
              <div className="form-group">
                <label>Task Title</label>
                <input type="text" value={newTitle} onChange={e => setNewTitle(e.target.value)} required />
              </div>
              <div className="form-group">
                <label>WBS Reference Code</label>
                <input type="text" value={newWBS} onChange={e => setNewWBS(e.target.value)} required />
              </div>
            </div>

            <div className="form-group">
              <label>Description</label>
              <textarea rows={3} value={newDesc} onChange={e => setNewDesc(e.target.value)} />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Assignee</label>
                <input type="text" placeholder="Staff Name" value={newAssignee} onChange={e => setNewAssignee(e.target.value)} />
              </div>
              <div className="form-group">
                <label>Priority</label>
                <select value={newPriority} onChange={e => setNewPriority(e.target.value)}>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                </select>
              </div>
            </div>

            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAdding(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Create Task</button>
            </div>
          </form>
        </div>
      )}

      <div className="kanban-cols">
        {columns.map(col => {
          const colTasks = getTasksByStatus(col);
          return (
            <div key={col} className="glass-panel kanban-col">
              <div className="kanban-col-header">
                <span style={{ fontWeight: '700', fontSize: '14px' }}>{col}</span>
                <span className="badge" style={{ background: 'rgba(255,255,255,0.06)' }}>{colTasks.length}</span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', flexGrow: 1, minHeight: '300px' }}>
                {colTasks.map(task => (
                  <div key={task.id} className="task-card">
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
                      <span style={{ fontSize: '11px', fontFamily: 'monospace', color: 'var(--accent-secondary)' }}>{task.wbsCode}</span>
                      <span className="badge" style={{
                        fontSize: '9px',
                        background: task.priority === 'High' ? 'rgba(239, 68, 68, 0.15)' : 'rgba(255,255,255,0.05)',
                        color: task.priority === 'High' ? 'var(--accent-danger)' : 'var(--text-secondary)'
                      }}>{task.priority}</span>
                    </div>

                    <h4 style={{ fontSize: '13px', fontWeight: '650', marginBottom: '4px' }}>{task.title}</h4>
                    <p style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '12px' }}>{task.description}</p>
                    
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>👤 {task.assignedTo || 'Unassigned'}</span>
                      <div style={{ display: 'flex', gap: '4px' }}>
                        {col !== 'Todo' && (
                          <button 
                            style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '11px' }}
                            onClick={() => handleMove(task, columns[columns.indexOf(col) - 1])}
                            title="Move back"
                          >
                            👈
                          </button>
                        )}
                        {col !== 'Done' && (
                          <button 
                            style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '11px' }}
                            onClick={() => handleMove(task, columns[columns.indexOf(col) + 1])}
                            title="Advance"
                          >
                            👉
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
