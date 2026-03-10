import { useState, useEffect } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
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
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    loadSessions();

    // Refresh session list when a new session is saved
    const cancel = EventsOn('session:saved', () => {
      loadSessions();
    });
    return () => { cancel(); };
  }, []);

  async function loadSessions() {
    try {
      if (window.go?.backend?.App) {
        const list = await window.go.backend.App.GetSessionList();
        setSessions(list || []);
      }
    } catch (e) {
      console.error('Failed to load sessions:', e);
    }
  }

  async function handleDelete(id: string, e: React.MouseEvent) {
    e.stopPropagation();
    try {
      await window.go.backend.App.DeleteSession(id);
      setSessions(prev => prev.filter(s => s.id !== id));
      if (expandedId === id) setExpandedId(null);
    } catch (e) {
      console.error('Failed to delete session:', e);
    }
  }

  function toggleExpand(id: string) {
    setExpandedId(expandedId === id ? null : id);
  }

  return (
    <div style={{ flex: 1, overflow: 'auto' }}>
      <div className="panel-header">
        <h2>Past Sessions</h2>
        <span className="session-count">{sessions.length} session{sessions.length !== 1 ? 's' : ''}</span>
      </div>

      {sessions.length === 0 ? (
        <div className="empty-state" style={{ height: 200 }}>
          No saved sessions yet
        </div>
      ) : (
        <div className="session-list">
          {sessions.map(sess => {
            const isExpanded = sess.id === expandedId;
            return (
              <div key={sess.id} className={`accordion-item ${isExpanded ? 'expanded' : ''}`}>
                <div
                  className="accordion-header"
                  onClick={() => toggleExpand(sess.id)}
                >
                  <div className="accordion-chevron">{isExpanded ? '\u25BC' : '\u25B6'}</div>
                  <div className="accordion-info">
                    <div className="session-title">{sess.title}</div>
                    <div className="session-meta">
                      {new Date(sess.created_at).toLocaleDateString()} &middot;{' '}
                      {formatDuration(sess.duration)} &middot;{' '}
                      {sess.segments?.length || 0} segments
                    </div>
                  </div>
                  <div className="accordion-actions">
                    <button
                      className="btn btn-secondary btn-sm"
                      onClick={(e) => { e.stopPropagation(); onExport(sess.id); }}
                    >
                      Export
                    </button>
                    <button
                      className="btn btn-secondary btn-sm btn-danger"
                      onClick={(e) => handleDelete(sess.id, e)}
                    >
                      Delete
                    </button>
                  </div>
                </div>

                {isExpanded && (
                  <div className="accordion-body">
                    {(!sess.segments || sess.segments.length === 0) ? (
                      <div className="empty-state" style={{ height: 80, fontSize: 13 }}>
                        No audio detected in this session
                      </div>
                    ) : (
                      sess.segments.map(seg => (
                        <div key={seg.id} className="segment">
                          <span className="timestamp">
                            [{Math.floor(seg.start_time / 60)}:{(Math.floor(seg.start_time) % 60).toString().padStart(2, '0')}]
                          </span>
                          <span className={`speaker ${seg.speaker.toLowerCase().replace(/\s+/g, '-')}`}>
                            {seg.speaker}:
                          </span>
                          <span className="text">{seg.text}</span>
                        </div>
                      ))
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
