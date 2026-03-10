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
    loadDevices();

    // Handle in-window keyboard shortcut for meeting toggle (Super+Shift+M)
    function handleKeyDown(e: KeyboardEvent) {
      if (e.metaKey && e.shiftKey && e.key === 'M') {
        e.preventDefault();
        if (isRecording) {
          handleStop();
        } else {
          handleStart();
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isRecording, micDevice, monitorDevice]);

  async function loadDevices() {
    try {
      if (window.go?.backend?.App) {
        const devs = await window.go.backend.App.ListAudioDevices();
        setDevices(devs || []);
        const mons = await window.go.backend.App.ListMonitorSources();
        setMonitors(mons || []);
        // Auto-select default monitor source (for speaker/system audio capture)
        if (mons && mons.length > 0 && !monitorDevice) {
          const def = mons.find((m: DeviceInfo) => m.IsDefault);
          setMonitorDevice(def ? def.Name : mons[0].Name);
        }
      }
    } catch (e) {
      console.error('Failed to load devices:', e);
    }
  }

  async function handleStart() {
    try {
      clearTranscript();
      await window.go.backend.App.StartSession(micDevice, monitorDevice);
    } catch (e) {
      console.error('Failed to start session:', e);
    }
  }

  async function handleStop(): Promise<Session | null> {
    try {
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
        <TranscriptPane segments={segments} isRecording={isRecording} />
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
