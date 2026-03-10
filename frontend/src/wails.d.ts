/* eslint-disable @typescript-eslint/no-explicit-any */
interface Window {
  go: {
    backend: {
      App: {
        ListAudioDevices(): Promise<any[]>;
        ListMonitorSources(): Promise<any[]>;
        StartSession(mic: string, monitor: string): Promise<void>;
        StopSession(): Promise<any>;
        GetSessionList(): Promise<any[]>;
        LoadSession(id: string): Promise<any>;
        ExportSession(id: string, format: string): Promise<string>;
        DeleteSession(id: string): Promise<void>;
        GetConfig(): Promise<any>;
        GetGPUInfo(): Promise<any>;
        GetModelStatus(): Promise<any>;
      };
    };
  };
}
