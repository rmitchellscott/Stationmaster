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
  sleep_enabled?: boolean;
  sleep_start_time?: string;
  sleep_end_time?: string;
  sleep_show_screen?: boolean;
  mirror_source_id?: string;
}

export const calculateBatteryPercentage = (voltage: number): number => {
  // Clamp voltage to valid range
  if (voltage <= 3.1) return 1;
  if (voltage >= 4.06) return 100;
  
  // Piecewise linear interpolation based on official API data
  const points = [
    [3.1, 1],
    [3.65, 54],
    [3.70, 58],
    [3.75, 62],
    [3.80, 66],
    [3.85, 70],
    [3.88, 73],
    [3.90, 75],
    [3.92, 76],
    [3.98, 81],
    [4.00, 90],
    [4.02, 95],
    [4.05, 95],
    [4.06, 100]
  ];
  
  // Find the two points to interpolate between
  for (let i = 0; i < points.length - 1; i++) {
    const [v1, p1] = points[i];
    const [v2, p2] = points[i + 1];
    
    if (voltage >= v1 && voltage <= v2) {
      // Linear interpolation between the two points
      const ratio = (voltage - v1) / (v2 - v1);
      return Math.round(p1 + ratio * (p2 - p1));
    }
  }
  
  return 1; // Fallback
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
    icon = <BatteryLow className="h-4 w-4" />;
    color = "";
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

export const isDeviceCurrentlySleeping = (device: Device, userTimezone: string): boolean => {
  if (!device.sleep_enabled || !device.sleep_start_time || !device.sleep_end_time) {
    return false;
  }

  // Get current time in user's timezone
  const now = new Date();
  const timeInTz = now.toLocaleTimeString('en-US', { 
    hour12: false, 
    timeZone: userTimezone,
    hour: '2-digit',
    minute: '2-digit'
  });
  
  const [targetHours, targetMinutes] = timeInTz.split(':').map(Number);
  const targetTimeMinutes = targetHours * 60 + targetMinutes;
  
  // Parse sleep times
  const [startHours, startMinutes] = device.sleep_start_time.split(':').map(Number);
  const [endHours, endMinutes] = device.sleep_end_time.split(':').map(Number);
  
  const sleepStartMinutes = startHours * 60 + startMinutes;
  const sleepEndMinutes = endHours * 60 + endMinutes;
  
  // Handle overnight sleep periods
  if (sleepStartMinutes > sleepEndMinutes) {
    return targetTimeMinutes >= sleepStartMinutes || targetTimeMinutes <= sleepEndMinutes;
  } else {
    return targetTimeMinutes >= sleepStartMinutes && targetTimeMinutes <= sleepEndMinutes;
  }
};