import { useEffect, useRef, useState, useCallback } from 'react';

export interface DeviceEvent {
  type: string;
  data: any;
}

export interface PlaylistIndexChangeEvent {
  device_id: string;
  current_index: number;
  current_item: any;
  active_items: any[];
  timestamp: string;
}

export interface DeviceEventsHookState {
  connected: boolean;
  error: string | null;
  lastEvent: DeviceEvent | null;
  currentIndex: number | null;
  currentItem: any | null;
  activeItems: any[];
  currentItemChanged: boolean;
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

  const connect = useCallback(() => {
    if (!deviceId || isManuallyDisconnectedRef.current) {
      return;
    }

    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    try {
      const eventSource = new EventSource(`/api/devices/${deviceId}/events`, {
        withCredentials: true,
      });

      eventSource.onopen = () => {
        console.log(`[SSE] Connected to device ${deviceId} events`);
        setState(prev => ({
          ...prev,
          connected: true,
          error: null,
        }));
      };

      eventSource.onerror = (error) => {
        console.error(`[SSE] Error for device ${deviceId}:`, error);
        setState(prev => ({
          ...prev,
          connected: false,
          error: 'Connection error',
        }));

        // Attempt to reconnect after 5 seconds unless manually disconnected
        if (!isManuallyDisconnectedRef.current) {
          reconnectTimeoutRef.current = setTimeout(() => {
            console.log(`[SSE] Attempting to reconnect to device ${deviceId}...`);
            connect();
          }, 5000);
        }
      };

      eventSource.onmessage = (event) => {
        try {
          const parsedEvent: DeviceEvent = JSON.parse(event.data);
          console.log(`[SSE] Received event for device ${deviceId}:`, parsedEvent);

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
              };
            });
          }
        } catch (parseError) {
          console.error(`[SSE] Failed to parse event data for device ${deviceId}:`, parseError);
        }
      };

      eventSourceRef.current = eventSource;
    } catch (connectionError) {
      console.error(`[SSE] Failed to connect to device ${deviceId}:`, connectionError);
      setState(prev => ({
        ...prev,
        connected: false,
        error: 'Failed to establish connection',
      }));
    }
  }, [deviceId]);

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