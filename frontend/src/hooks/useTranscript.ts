import { useState, useEffect, useCallback } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { Segment } from '../types';

export function useTranscript() {
  const [segments, setSegments] = useState<Segment[]>([]);

  useEffect(() => {
    const cancel = EventsOn('transcript:segment', (seg: Segment) => {
      setSegments(prev => [...prev, seg]);
    });

    return () => {
      cancel();
    };
  }, []);

  const clear = useCallback(() => {
    setSegments([]);
  }, []);

  return { segments, clear };
}
