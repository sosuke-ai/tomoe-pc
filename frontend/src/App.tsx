import { useState, useEffect } from 'react'
import TranscriptPane from './components/TranscriptPane'
import SourceSelector from './components/SourceSelector'
import SessionControls from './components/SessionControls'
import SessionList from './components/SessionList'
import SettingsPanel from './components/SettingsPanel'
import StatusBar from './components/StatusBar'
import ExportDialog from './components/ExportDialog'
import { useTranscript } from './hooks/useTranscript'
import { useSession } from './hooks/useSession'
import { DeviceInfo, Session } from './types'

type View = 'live' | 'sessions' | 'settings';

function App() {
  const [view, setView] = useState<View>('live');
  const [micDevice, setMicDevice] = useState('default');
  const [monitorDevice, setMonitorDevice] = useState('');
  const [devices, setDevices] = useState<DeviceInfo[]>([]);
  const [monitors, setMonitors] = useState<DeviceInfo[]>([]);
  const [exportSessionId, setExportSessionId] = useState<string | null>(null);
  const { segments, clear: clearTranscript } = useTranscript();
  const { isRecording, startTime, reset: resetSession } = useSession();

  useEffect(() => {
    // Load devices on mount
    loadDevices();
  }, []);

  async function loadDevices() {
    try {
      // @ts-ignore - Wails bindings are injected at runtime
      if (window.go?.backend?.App) {
        const devs = await window.go.backend.App.ListAudioDevices();
        setDevices(devs || []);
        const mons = await window.go.backend.App.ListMonitorSources();
        setMonitors(mons || []);
      }
    } catch (e) {
      console.error('Failed to load devices:', e);
    }
  }

  async function handleStart() {
    try {
      clearTranscript();
      // @ts-ignore
      await window.go.backend.App.StartSession(micDevice, monitorDevice);
    } catch (e) {
      console.error('Failed to start session:', e);
    }
  }

  async function handleStop(): Promise<Session | null> {
    try {
      // @ts-ignore
      const sess = await window.go.backend.App.StopSession();
      resetSession();
      return sess;
    } catch (e) {
      console.error('Failed to stop session:', e);
      return null;
    }
  }

  return (
    <div className="app">
      <div className="toolbar">
        <SourceSelector
          devices={devices}
          monitors={monitors}
          micDevice={micDevice}
          monitorDevice={monitorDevice}
          onMicChange={setMicDevice}
          onMonitorChange={setMonitorDevice}
          disabled={isRecording}
        />
        <div className="spacer" />
        <SessionControls
          isRecording={isRecording}
          segmentCount={segments.length}
          onStart={handleStart}
          onStop={handleStop}
        />
        <button
          className="btn-icon"
          title="Past Sessions"
          onClick={() => setView(view === 'sessions' ? 'live' : 'sessions')}
        >
          &#x1F4CB;
        </button>
        <button
          className="btn-icon"
          title="Settings"
          onClick={() => setView(view === 'settings' ? 'live' : 'settings')}
        >
          &#x2699;
        </button>
      </div>

      {view === 'live' && (
        <TranscriptPane segments={segments} />
      )}

      {view === 'sessions' && (
        <SessionList onExport={setExportSessionId} />
      )}

      {view === 'settings' && (
        <SettingsPanel />
      )}

      <StatusBar
        isRecording={isRecording}
        startTime={startTime}
        segmentCount={segments.length}
      />

      {exportSessionId && (
        <ExportDialog
          sessionId={exportSessionId}
          onClose={() => setExportSessionId(null)}
        />
      )}
    </div>
  )
}

export default App
