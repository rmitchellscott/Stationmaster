import {
  Battery,
  BatteryFull,
  BatteryMedium,
  BatteryLow,
  BatteryWarning,
  Wifi,
  WifiOff,
} from "lucide-react";

export interface Device {
  id: string;
  user_id?: string;
  mac_address: string;
  friendly_id: string;
  name?: string;
  api_key: string;
  is_claimed: boolean;
  firmware_version?: string;
  battery_voltage?: number;
  rssi?: number;
  refresh_rate: number;
  last_seen?: string;
  last_playlist_index: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export const calculateBatteryPercentage = (voltage: number): number => {
  if (voltage >= 3.7) return Math.min(100, Math.round(75 + ((voltage - 3.7) / (4.2 - 3.7)) * 25));
  if (voltage >= 3.4) return Math.round(50 + ((voltage - 3.4) / (3.7 - 3.4)) * 25);
  if (voltage >= 3.0) return Math.round(25 + ((voltage - 3.0) / (3.4 - 3.0)) * 25);
  return Math.round((voltage - 2.75) / (3.0 - 2.75) * 25);
};

export const getSignalQuality = (rssi: number): { quality: string; strength: number; color: string } => {
  if (rssi > -50) return { quality: "Excellent", strength: 5, color: "" };
  if (rssi > -60) return { quality: "Good", strength: 4, color: "" };
  if (rssi > -70) return { quality: "Fair", strength: 3, color: "" };
  if (rssi > -80) return { quality: "Poor", strength: 2, color: "text-destructive" };
  return { quality: "Very Poor", strength: 1, color: "text-destructive" };
};

export const getBatteryDisplay = (voltage?: number) => {
  if (!voltage) {
    return {
      icon: <Battery className="h-4 w-4 text-muted-foreground" />,
      text: "N/A",
      tooltip: "Battery status unknown"
    };
  }
  
  const percentage = calculateBatteryPercentage(voltage);
  let icon;
  let color;
  
  if (percentage > 75) {
    icon = <BatteryFull className="h-4 w-4" />;
    color = "";
  } else if (percentage > 50) {
    icon = <BatteryMedium className="h-4 w-4" />;
    color = "";
  } else if (percentage > 25) {
    icon = <BatteryLow className="h-4 w-4 text-destructive" />;
    color = "text-destructive";
  } else {
    icon = <BatteryWarning className="h-4 w-4 text-destructive" />;
    color = "text-destructive";
  }
  
  return {
    icon,
    text: `${percentage}%`,
    tooltip: `Battery Level: ${percentage}% (${voltage.toFixed(1)}V)`,
    color
  };
};

export const getSignalDisplay = (rssi?: number) => {
  if (!rssi) {
    return {
      icon: <WifiOff className="h-4 w-4 text-muted-foreground" />,
      text: "N/A",
      tooltip: "Signal strength unknown"
    };
  }
  
  const { quality, color } = getSignalQuality(rssi);
  
  return {
    icon: <Wifi className={`h-4 w-4 ${color}`} />,
    text: quality,
    tooltip: `Signal Quality: ${quality} (${rssi}dBm)`,
    color
  };
};

export const getDeviceStatus = (device: Device): string => {
  if (!device.is_active) return "inactive";
  
  if (!device.last_seen) {
    return "never_connected";
  }
  
  const lastSeenTime = new Date(device.last_seen).getTime();
  const now = Date.now();
  const timeDiff = now - lastSeenTime;
  
  // Online if seen within last 10 minutes
  if (timeDiff <= 10 * 60 * 1000) {
    return "online";
  }
  
  // Recently online if seen within last hour
  if (timeDiff <= 60 * 60 * 1000) {
    return "recently_online";
  }
  
  return "offline";
};