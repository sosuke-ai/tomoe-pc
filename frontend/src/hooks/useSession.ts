import { useState, useEffect, useCallback } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export function useSession() {
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [isRecording, setIsRecording] = useState(false);
  const [startTime, setStartTime] = useState<Date | null>(null);

  useEffect(() => {
    const cancelStart = EventsOn('session:started', (id: string) => {
      setSessionId(id);
      setIsRecording(true);
      setStartTime(new Date());
    });

    const cancelStop = EventsOn('session:stopped', () => {
      setIsRecording(false);
    });

    const cancelSaved = EventsOn('session:saved', () => {
      // Session was saved
    });

    return () => {
      cancelStart();
      cancelStop();
      cancelSaved();
    };
  }, []);

  const reset = useCallback(() => {
    setSessionId(null);
    setIsRecording(false);
    setStartTime(null);
  }, []);

  return { sessionId, isRecording, startTime, reset };
}
