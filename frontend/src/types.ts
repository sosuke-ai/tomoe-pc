export interface Segment {
  id: string;
  speaker: string;
  text: string;
  start_time: number;
  end_time: number;
  source: string;
}

export interface Session {
  id: string;
  title: string;
  created_at: string;
  ended_at?: string;
  duration: number;
  sources: string[];
  segments: Segment[];
  audio_path?: string;
}

export interface DeviceInfo {
  ID: string;
  Name: string;
  IsDefault: boolean;
  DeviceType: number; // 0=Input, 1=Monitor
}

export interface Config {
  hotkey: {
    binding: string;
    meeting_binding: string;
  };
  audio: {
    device: string;
  };
  transcription: {
    gpu_enabled: boolean;
    model_path: string;
  };
  output: {
    auto_paste: boolean;
    clipboard: boolean;
  };
  meeting: {
    default_sources: string;
    monitor_device: string;
    speaker_threshold: number;
    max_speech_duration: number;
    min_silence_duration: number;
    auto_save: boolean;
  };
}

export interface GPUInfo {
  Available: boolean;
  Sufficient: boolean;
  Name: string;
  VRAM: number;
  DriverVersion: string;
}

export interface ModelStatus {
  ParakeetReady: boolean;
  VADReady: boolean;
  SpeakerEmbeddingReady: boolean;
  ModelDir: string;
}
