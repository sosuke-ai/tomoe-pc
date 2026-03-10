import { useEffect, useRef } from 'react';
import { Segment } from '../types';

interface Props {
  segments: Segment[];
}

function formatTime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

function speakerClass(speaker: string): string {
  if (speaker === 'You') return 'you';
  if (speaker === 'Person 1') return 'person-1';
  if (speaker === 'Person 2') return 'person-2';
  if (speaker === 'Person 3') return 'person-3';
  return 'other';
}

export default function TranscriptPane({ segments }: Props) {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [segments]);

  if (segments.length === 0) {
    return (
      <div className="transcript-pane">
        <div className="empty-state">
          Select audio sources and click Start to begin transcription
        </div>
      </div>
    );
  }

  return (
    <div className="transcript-pane">
      {segments.map((seg) => (
        <div key={seg.id} className="segment">
          <span className="timestamp">[{formatTime(seg.start_time)}]</span>
          <span className={`speaker ${speakerClass(seg.speaker)}`}>
            {seg.speaker}:
          </span>
          <span className="text">{seg.text}</span>
        </div>
      ))}
      <div ref={endRef} />
    </div>
  );
}
