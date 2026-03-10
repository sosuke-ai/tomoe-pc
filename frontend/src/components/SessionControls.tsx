interface Props {
  isRecording: boolean;
  segmentCount: number;
  onStart: () => void;
  onStop: () => void;
}

export default function SessionControls({ isRecording, onStart, onStop }: Props) {
  return (
    <button
      className={`btn ${isRecording ? 'btn-primary' : 'btn-secondary'}`}
      onClick={isRecording ? onStop : onStart}
    >
      {isRecording ? '■ Stop' : '● Start'}
    </button>
  );
}
