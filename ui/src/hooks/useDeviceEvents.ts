import { useEffect, useRef, useState, useCallback } from 'react';

export interface DeviceEvent {
  type: string;
  data: any;
}

export interface SleepConfig {
  enabled: boolean;
  start_time: string;
  end_time: string;
  show_screen: boolean;
  currently_sleeping: boolean;
  sleep_screen_served: boolean;
}

export interface PlaylistIndexChangeEvent {
  device_id: string;
  current_index: number;
  current_item: any;
  active_items: any[];
  timestamp: string;
  sleep_config: SleepConfig;
}

export interface DeviceEventsHookState {
  connected: boolean;
  error: string | null;
  lastEvent: DeviceEvent | null;
  currentIndex: number | null;
  currentItem: any | null;
  activeItems: any[];
  currentItemChanged: boolean;
  sleepConfig: SleepConfig | null;
}

export interface DeviceEventsHookResult extends DeviceEventsHookState {
  disconnect: () => void;
  reconnect: () => void;
  clearCurrentItemChanged: () => void;
}

export function useDeviceEvents(deviceId: string): DeviceEventsHookResult {
  const [state, setState] = useState<DeviceEventsHookState>({
    connected: false,
    error: null,
    lastEvent: null,
    currentIndex: null,
    currentItem: null,
    activeItems: [],
    currentItemChanged: false,
    sleepConfig: null,
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const isManuallyDisconnectedRef = useRef(false);

  const disconnect = useCallback(() => {
    isManuallyDisconnectedRef.current = true;
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    setState(prev => ({
      ...prev,
      connected: false,
      error: null,
    }));
  }, []);

  // Fetch initial device state to populate sleep config before SSE connection
  const fetchInitialState = useCallback(async () => {
    if (!deviceId) return;

    try {
      const response = await fetch(`/api/devices/${deviceId}/active-items`, {
        credentials: 'include',
      });
      
      if (response.ok) {
        const data = await response.json();
        setState(prev => ({
          ...prev,
          currentIndex: data.current_index || null,
          currentItem: data.currently_showing || null,
          activeItems: data.active_items || [],
          sleepConfig: data.sleep_config || null,
        }));
      }
    } catch (error) {
      // Silently fail - SSE will provide updates when available
    }
  }, [deviceId]);

  const connect = useCallback(() => {
    if (!deviceId || isManuallyDisconnectedRef.current) {
      return;
    }

    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Fetch initial state first
    fetchInitialState();

    try {
      const eventSource = new EventSource(`/api/devices/${deviceId}/events`, {
        withCredentials: true,
      });

      eventSource.onopen = () => {
        setState(prev => ({
          ...prev,
          connected: true,
          error: null,
        }));
      };

      eventSource.onerror = (error) => {
        setState(prev => ({
          ...prev,
          connected: false,
          error: 'Connection error',
        }));

        // Attempt to reconnect after 5 seconds unless manually disconnected
        if (!isManuallyDisconnectedRef.current) {
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, 5000);
        }
      };

      eventSource.onmessage = (event) => {
        try {
          const parsedEvent: DeviceEvent = JSON.parse(event.data);

          setState(prev => ({
            ...prev,
            lastEvent: parsedEvent,
          }));

          // Handle specific event types
          if (parsedEvent.type === 'playlist_index_changed') {
            const data = parsedEvent.data as PlaylistIndexChangeEvent;
            setState(prev => {
              const currentItemChanged = prev.currentItem?.id !== data.current_item?.id;
              return {
                ...prev,
                currentIndex: data.current_index,
                currentItem: data.current_item,
                activeItems: data.active_items || [],
                currentItemChanged,
                sleepConfig: data.sleep_config || null,
              };
            });
          } else if (parsedEvent.type === 'device_settings_updated') {
            // Handle device settings updates (like sleep config changes)
            const data = parsedEvent.data;
            setState(prev => ({
              ...prev,
              sleepConfig: data.sleep_config || null,
            }));
          }
        } catch (parseError) {
        }
      };

      eventSourceRef.current = eventSource;
    } catch (connectionError) {
      setState(prev => ({
        ...prev,
        connected: false,
        error: 'Failed to establish connection',
      }));
    }
  }, [deviceId, fetchInitialState]);

  const reconnect = useCallback(() => {
    isManuallyDisconnectedRef.current = false;
    disconnect();
    // Small delay to ensure cleanup is complete
    setTimeout(connect, 100);
  }, [connect, disconnect]);

  const clearCurrentItemChanged = useCallback(() => {
    setState(prev => ({
      ...prev,
      currentItemChanged: false,
    }));
  }, []);

  // Connect when device ID changes or component mounts
  useEffect(() => {
    if (deviceId) {
      isManuallyDisconnectedRef.current = false;
      connect();
    }

    return () => {
      disconnect();
    };
  }, [deviceId, connect, disconnect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  return {
    ...state,
    disconnect,
    reconnect,
    clearCurrentItemChanged,
  };
}

export default useDeviceEvents;