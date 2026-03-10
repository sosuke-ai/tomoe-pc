import { DeviceInfo } from '../types';

interface Props {
  devices: DeviceInfo[];
  monitors: DeviceInfo[];
  micDevice: string;
  monitorDevice: string;
  onMicChange: (device: string) => void;
  onMonitorChange: (device: string) => void;
  disabled: boolean;
}

export default function SourceSelector({
  devices, monitors, micDevice, monitorDevice,
  onMicChange, onMonitorChange, disabled,
}: Props) {
  return (
    <>
      <select
        value={micDevice}
        onChange={(e) => onMicChange(e.target.value)}
        disabled={disabled}
        title="Microphone"
      >
        <option value="">No Mic</option>
        <option value="default">Default Mic</option>
        {devices
          .filter(d => d.DeviceType === 0)
          .map(d => (
            <option key={d.ID} value={d.Name}>
              {d.Name}{d.IsDefault ? ' *' : ''}
            </option>
          ))
        }
      </select>

      <select
        value={monitorDevice}
        onChange={(e) => onMonitorChange(e.target.value)}
        disabled={disabled}
        title="System Audio"
      >
        <option value="">No System Audio</option>
        {monitors.map(d => (
          <option key={d.ID} value={d.Name}>
            {d.Name}{d.IsDefault ? ' *' : ''}
          </option>
        ))}
      </select>
    </>
  );
}
