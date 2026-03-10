import { useState, useEffect } from 'react';
import { Session } from '../types';

interface Props {
  onExport: (sessionId: string) => void;
}

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}m${s.toString().padStart(2, '0')}s`;
}

export default function SessionList({ onExport }: Props) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  useEffect(() => {
    loadSessions();
  }, []);

  async function loadSessions() {
    try {
      const list = await window.go.backend.App.GetSessionList();
      setSessions(list || []);
    } catch (e) {
      console.error('Failed to load sessions:', e);
    }
  }

  async function handleDelete(id: string) {
    try {
      await window.go.backend.App.DeleteSession(id);
      setSessions(prev => prev.filter(s => s.id !== id));
      if (selectedId === id) setSelectedId(null);
    } catch (e) {
      console.error('Failed to delete session:', e);
    }
  }

  const selected = sessions.find(s => s.id === selectedId);

  return (
    <div style={{ flex: 1, overflow: 'auto' }}>
      <div className="panel-header">
        <h2>Past Sessions</h2>
      </div>

      {sessions.length === 0 ? (
        <div className="empty-state" style={{ height: 200 }}>
          No saved sessions yet
        </div>
      ) : (
        <div className="session-list">
          {sessions.map(sess => (
            <div
              key={sess.id}
              className="session-item"
              onClick={() => setSelectedId(sess.id === selectedId ? null : sess.id)}
            >
              <div>
                <div className="session-title">{sess.title}</div>
                <div className="session-meta">
                  {new Date(sess.created_at).toLocaleDateString()} &middot;{' '}
                  {formatDuration(sess.duration)} &middot;{' '}
                  {sess.segments?.length || 0} segments
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                <button
                  className="btn btn-secondary"
                  onClick={(e) => { e.stopPropagation(); onExport(sess.id); }}
                >
                  Export
                </button>
                <button
                  className="btn btn-secondary"
                  onClick={(e) => { e.stopPropagation(); handleDelete(sess.id); }}
                  style={{ color: '#e94560' }}
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {selected && (
        <div className="transcript-pane" style={{ maxHeight: 300, borderTop: '1px solid var(--border)' }}>
          {selected.segments?.map(seg => (
            <div key={seg.id} className="segment">
              <span className="timestamp">
                [{Math.floor(seg.start_time / 60)}:{(Math.floor(seg.start_time) % 60).toString().padStart(2, '0')}]
              </span>
              <span className="speaker">{seg.speaker}:</span>
              <span className="text">{seg.text}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
