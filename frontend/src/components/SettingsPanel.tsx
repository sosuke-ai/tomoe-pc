import { useState, useEffect } from 'react';
import { Config, GPUInfo, ModelStatus } from '../types';

export default function SettingsPanel() {
  const [config, setConfig] = useState<Config | null>(null);
  const [gpuInfo, setGpuInfo] = useState<GPUInfo | null>(null);
  const [modelStatus, setModelStatus] = useState<ModelStatus | null>(null);

  useEffect(() => {
    loadInfo();
  }, []);

  async function loadInfo() {
    try {
      // @ts-ignore
      if (window.go?.backend?.App) {
        const cfg = await window.go.backend.App.GetConfig();
        setConfig(cfg);
        const gpu = await window.go.backend.App.GetGPUInfo();
        setGpuInfo(gpu);
        const models = await window.go.backend.App.GetModelStatus();
        setModelStatus(models);
      }
    } catch (e) {
      console.error('Failed to load settings:', e);
    }
  }

  return (
    <div style={{ flex: 1, overflow: 'auto' }}>
      <div className="panel-header">
        <h2>Settings</h2>
      </div>
      <div className="settings-panel">
        <h3>GPU</h3>
        {gpuInfo && (
          <>
            <div className="setting-row">
              <label>Available</label>
              <span className="value">{gpuInfo.Available ? 'Yes' : 'No'}</span>
            </div>
            {gpuInfo.Available && (
              <>
                <div className="setting-row">
                  <label>GPU</label>
                  <span className="value">{gpuInfo.Name}</span>
                </div>
                <div className="setting-row">
                  <label>VRAM</label>
                  <span className="value">{gpuInfo.VRAM} MB</span>
                </div>
                <div className="setting-row">
                  <label>Driver</label>
                  <span className="value">{gpuInfo.DriverVersion}</span>
                </div>
              </>
            )}
            <div className="setting-row">
              <label>GPU Enabled</label>
              <span className="value">{config?.transcription.gpu_enabled ? 'Yes' : 'No'}</span>
            </div>
          </>
        )}

        <h3 style={{ marginTop: 16 }}>Models</h3>
        {modelStatus && (
          <>
            <div className="setting-row">
              <label>Parakeet TDT</label>
              <span className="value" style={{ color: modelStatus.ParakeetReady ? '#4ec9b0' : '#e94560' }}>
                {modelStatus.ParakeetReady ? 'Ready' : 'Not Downloaded'}
              </span>
            </div>
            <div className="setting-row">
              <label>Silero VAD</label>
              <span className="value" style={{ color: modelStatus.VADReady ? '#4ec9b0' : '#e94560' }}>
                {modelStatus.VADReady ? 'Ready' : 'Not Downloaded'}
              </span>
            </div>
            <div className="setting-row">
              <label>Speaker Embedding</label>
              <span className="value" style={{ color: modelStatus.SpeakerEmbeddingReady ? '#4ec9b0' : '#e94560' }}>
                {modelStatus.SpeakerEmbeddingReady ? 'Ready' : 'Not Downloaded'}
              </span>
            </div>
            <div className="setting-row">
              <label>Model Directory</label>
              <span className="value" style={{ fontSize: 11 }}>{modelStatus.ModelDir}</span>
            </div>
          </>
        )}

        <h3 style={{ marginTop: 16 }}>Hotkeys</h3>
        {config && (
          <>
            <div className="setting-row">
              <label>Dictation</label>
              <span className="value">{config.hotkey.binding}</span>
            </div>
            <div className="setting-row">
              <label>Meeting Toggle</label>
              <span className="value">{config.hotkey.meeting_binding}</span>
            </div>
          </>
        )}

        <h3 style={{ marginTop: 16 }}>Meeting</h3>
        {config && (
          <>
            <div className="setting-row">
              <label>Default Sources</label>
              <span className="value">{config.meeting.default_sources}</span>
            </div>
            <div className="setting-row">
              <label>Speaker Threshold</label>
              <span className="value">{config.meeting.speaker_threshold}</span>
            </div>
            <div className="setting-row">
              <label>Auto-Save</label>
              <span className="value">{config.meeting.auto_save ? 'Yes' : 'No'}</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
