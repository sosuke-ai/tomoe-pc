import { useState, useEffect } from 'react';

interface Props {
  isRecording: boolean;
  startTime: Date | null;
  segmentCount: number;
}

function formatElapsed(start: Date): string {
  const elapsed = Math.floor((Date.now() - start.getTime()) / 1000);
  const m = Math.floor(elapsed / 60);
  const s = elapsed % 60;
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

export default function StatusBar({ isRecording, startTime, segmentCount }: Props) {
  const [, setTick] = useState(0);

  useEffect(() => {
    if (!isRecording) return;
    const interval = setInterval(() => setTick(t => t + 1), 1000);
    return () => clearInterval(interval);
  }, [isRecording]);

  return (
    <div className="status-bar">
      {isRecording ? (
        <>
          <div className="recording-indicator">
            <div className="recording-dot" />
            Recording
          </div>
          <span>{startTime ? formatElapsed(startTime) : '00:00'}</span>
          <span>{segmentCount} segment{segmentCount !== 1 ? 's' : ''}</span>
        </>
      ) : (
        <span>Ready</span>
      )}
    </div>
  );
}
