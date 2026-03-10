import { useState } from 'react';

interface Props {
  sessionId: string;
  onClose: () => void;
}

const formats = [
  { id: 'markdown', label: 'Markdown (.md)', desc: 'Formatted text with speaker labels' },
  { id: 'text', label: 'Plain Text (.txt)', desc: 'Simple text format' },
  { id: 'srt', label: 'SRT Subtitles (.srt)', desc: 'Standard subtitle format' },
];

export default function ExportDialog({ sessionId, onClose }: Props) {
  const [selectedFormat, setSelectedFormat] = useState('markdown');
  const [exported, setExported] = useState(false);

  async function handleExport() {
    try {
      const content = await window.go.backend.App.ExportSession(sessionId, selectedFormat);
      // Copy to clipboard
      await navigator.clipboard.writeText(content);
      setExported(true);
      setTimeout(() => setExported(false), 2000);
    } catch (e) {
      console.error('Export failed:', e);
    }
  }

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={e => e.stopPropagation()}>
        <h3>Export Session</h3>
        <div className="format-options">
          {formats.map(fmt => (
            <div
              key={fmt.id}
              className={`format-option ${selectedFormat === fmt.id ? 'selected' : ''}`}
              onClick={() => setSelectedFormat(fmt.id)}
            >
              <strong>{fmt.label}</strong>
              <div style={{ fontSize: 12, color: 'var(--text-dim)', marginTop: 2 }}>
                {fmt.desc}
              </div>
            </div>
          ))}
        </div>
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button className="btn btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={handleExport}>
            {exported ? 'Copied!' : 'Export & Copy'}
          </button>
        </div>
      </div>
    </div>
  );
}
