import React from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Device, getBatteryDisplay, getSignalDisplay } from "@/utils/deviceHelpers";
import { Monitor } from "lucide-react";

interface DeviceSelectorProps {
  devices: Device[];
  selectedDeviceId: string | null;
  onDeviceChange: (deviceId: string) => void;
  loading?: boolean;
}

export function DeviceSelector({ 
  devices, 
  selectedDeviceId, 
  onDeviceChange,
  loading = false 
}: DeviceSelectorProps) {
  const selectedDevice = devices.find(d => d.id === selectedDeviceId);

  if (loading) {
    return (
      <div className="flex items-center gap-3 p-4 border rounded-lg bg-muted/50">
        <Monitor className="h-5 w-5 text-muted-foreground" />
        <span className="text-muted-foreground">Loading devices...</span>
      </div>
    );
  }

  if (devices.length === 0) {
    return (
      <div className="flex items-center gap-3 p-4 border rounded-lg bg-muted/50">
        <Monitor className="h-5 w-5 text-muted-foreground" />
        <span className="text-muted-foreground">No devices found. Add a device in Settings to get started.</span>
      </div>
    );
  }

  const DeviceInfo = ({ device, compact = false }: { device: Device; compact?: boolean }) => {
    const battery = getBatteryDisplay(device.battery_voltage);
    const signal = getSignalDisplay(device.rssi);
    
    return (
      <div className="flex items-center gap-3">
        <Monitor className="h-4 w-4 text-muted-foreground" />
        <div className="flex-1">
          <div className="font-medium">
            {device.name || device.friendly_id}
          </div>
          {!compact && (
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <div className="flex items-center gap-1">
                {battery.icon}
                <span className={battery.color}>{battery.text}</span>
              </div>
              <div className="flex items-center gap-1">
                {signal.icon}
                <span className={signal.color}>{signal.text}</span>
              </div>
            </div>
          )}
        </div>
        {compact && (
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1">
              {battery.icon}
              <span className={`text-sm ${battery.color}`}>{battery.text}</span>
            </div>
            <div className="flex items-center gap-1">
              {signal.icon}
              <span className={`text-sm ${signal.color}`}>{signal.text}</span>
            </div>
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">TRMNL</label>
      <Select value={selectedDeviceId || ""} onValueChange={onDeviceChange}>
        <SelectTrigger className="w-full">
          <SelectValue>
            {selectedDevice ? (
              <DeviceInfo device={selectedDevice} compact={true} />
            ) : (
              <span className="text-muted-foreground">Select a device</span>
            )}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          {devices.map((device) => (
            <SelectItem key={device.id} value={device.id}>
              <DeviceInfo device={device} />
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {selectedDevice && (
        <div className="text-xs text-muted-foreground">
          ID: {selectedDevice.friendly_id} • Last seen: {
            selectedDevice.last_seen 
              ? new Date(selectedDevice.last_seen).toLocaleString()
              : "Never"
          }
        </div>
      )}
    </div>
  );
}
