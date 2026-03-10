import { useState, useEffect } from 'react';
import { Config, GPUInfo, ModelStatus } from '../types';

export default function SettingsPanel() {
  const [config, setConfig] = useState<Config | null>(null);
  const [gpuInfo, setGpuInfo] = useState<GPUInfo | null>(null);
  const [modelStatus, setModelStatus] = useState<ModelStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadInfo();
  }, []);

  async function loadInfo() {
    try {
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
      setError(String(e));
    }
  }

  if (error) {
    return (
      <div style={{ flex: 1, overflow: 'auto' }}>
        <div className="panel-header"><h2>Settings</h2></div>
        <div className="settings-panel">
          <p style={{ color: '#e94560' }}>Failed to load settings: {error}</p>
        </div>
      </div>
    );
  }

  return (
    <div style={{ flex: 1, overflow: 'auto' }}>
      <div className="panel-header">
        <h2>Settings</h2>
      </div>
      <div className="settings-panel">
        <h3>GPU</h3>
        {gpuInfo ? (
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
                  <span className="value">{gpuInfo.VRAMMB} MB</span>
                </div>
                <div className="setting-row">
                  <label>CUDA</label>
                  <span className="value">{gpuInfo.CUDAVersion}</span>
                </div>
              </>
            )}
            <div className="setting-row">
              <label>GPU Enabled</label>
              <span className="value">{config?.Transcription?.GPUEnabled ? 'Yes' : 'No'}</span>
            </div>
          </>
        ) : (
          <p style={{ color: 'var(--text-dim)', fontSize: 13 }}>Loading...</p>
        )}

        <h3 style={{ marginTop: 16 }}>Models</h3>
        {modelStatus ? (
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
        ) : (
          <p style={{ color: 'var(--text-dim)', fontSize: 13 }}>Loading...</p>
        )}

        <h3 style={{ marginTop: 16 }}>Hotkeys</h3>
        {config && (
          <>
            <div className="setting-row">
              <label>Dictation</label>
              <span className="value">{config.Hotkey?.Binding}</span>
            </div>
            <div className="setting-row">
              <label>Meeting Toggle</label>
              <span className="value">{config.Hotkey?.MeetingBinding}</span>
            </div>
          </>
        )}

        <h3 style={{ marginTop: 16 }}>Meeting</h3>
        {config && (
          <>
            <div className="setting-row">
              <label>Default Sources</label>
              <span className="value">{config.Meeting?.DefaultSources}</span>
            </div>
            <div className="setting-row">
              <label>Speaker Threshold</label>
              <span className="value">{config.Meeting?.SpeakerThreshold}</span>
            </div>
            <div className="setting-row">
              <label>Auto-Save</label>
              <span className="value">{config.Meeting?.AutoSave ? 'Yes' : 'No'}</span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
