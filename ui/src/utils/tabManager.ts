interface TabMessage {
  type: string
  data: any
  timestamp: number
  tabId: string
}

interface DeviceUpdate {
  device_id: string
  battery_voltage?: number
  rssi?: number
  firmware_version?: string
  last_seen?: string
  is_active?: boolean
}

class TabManager {
  private tabId: string
  private channel: BroadcastChannel | null = null
  private isPrimary: boolean = false
  private heartbeatInterval: NodeJS.Timeout | null = null
  private listeners: Map<string, Set<(data: any) => void>> = new Map()
  private sseConnections: Map<string, EventSource> = new Map()

  constructor() {
    this.tabId = Math.random().toString(36).substr(2, 9)
    
    if (typeof window !== 'undefined' && 'BroadcastChannel' in window) {
      this.channel = new BroadcastChannel('stationmaster-tabs')
      this.setupMessageHandling()
      this.electPrimary()
    } else {
      // Fallback: assume primary if BroadcastChannel not supported
      this.isPrimary = true
    }
  }

  private setupMessageHandling() {
    if (!this.channel) return

    this.channel.addEventListener('message', (event) => {
      const message: TabMessage = event.data
      
      // Ignore messages from this tab
      if (message.tabId === this.tabId) return

      switch (message.type) {
        case 'primary_election':
          this.handlePrimaryElection(message)
          break
        case 'primary_heartbeat':
          this.handlePrimaryHeartbeat(message)
          break
        case 'device_update':
          this.handleDeviceUpdate(message.data)
          break
        case 'auth_change':
          this.handleAuthChange(message.data)
          break
        case 'logout':
          this.handleLogout()
          break
      }
    })

    // Clean up on page unload
    window.addEventListener('beforeunload', () => {
      this.cleanup()
    })
  }

  private electPrimary() {
    if (!this.channel) return

    // Send election message
    this.broadcast('primary_election', { tabId: this.tabId, timestamp: Date.now() })

    // Wait a bit to see if there's already a primary
    setTimeout(() => {
      if (!this.isPrimary) {
        this.becomePrimary()
      }
    }, 100)
  }

  private becomePrimary() {
    this.isPrimary = true

    // Start heartbeat
    this.heartbeatInterval = setInterval(() => {
      this.broadcast('primary_heartbeat', { tabId: this.tabId })
    }, 5000)

    // Notify listeners that this tab is now primary
    this.emit('primary_changed', true)
  }

  private handlePrimaryElection(message: TabMessage) {
    const { tabId: candidateTabId, timestamp } = message.data
    
    // If we're not primary and this candidate is newer, let them be primary
    if (!this.isPrimary) return

    // If we're primary but this candidate has a newer timestamp, step down
    const ourTimestamp = Date.now()
    if (timestamp > ourTimestamp) {
      this.stepDownFromPrimary()
    } else {
      // We're staying primary, send heartbeat to assert dominance
      this.broadcast('primary_heartbeat', { tabId: this.tabId })
    }
  }

  private handlePrimaryHeartbeat(message: TabMessage) {
    const { tabId: primaryTabId } = message.data
    
    if (primaryTabId !== this.tabId && this.isPrimary) {
      // Another tab claims to be primary, step down
      this.stepDownFromPrimary()
    }
  }

  private stepDownFromPrimary() {
    this.isPrimary = false
    
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
      this.heartbeatInterval = null
    }

    // Close all SSE connections
    this.sseConnections.forEach(source => source.close())
    this.sseConnections.clear()

    this.emit('primary_changed', false)
  }

  private handleDeviceUpdate(data: DeviceUpdate) {
    this.emit('device_update', data)
  }

  private handleAuthChange(data: any) {
    this.emit('auth_change', data)
  }

  private handleLogout() {
    this.emit('logout', {})
  }

  // Public API
  public getTabId(): string {
    return this.tabId
  }

  public isPrimaryTab(): boolean {
    return this.isPrimary
  }

  public addListener(event: string, callback: (data: any) => void) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)
  }

  public removeListener(event: string, callback: (data: any) => void) {
    const eventListeners = this.listeners.get(event)
    if (eventListeners) {
      eventListeners.delete(callback)
    }
  }

  private emit(event: string, data: any) {
    const eventListeners = this.listeners.get(event)
    if (eventListeners) {
      eventListeners.forEach(callback => callback(data))
    }
  }

  public broadcast(type: string, data: any) {
    if (!this.channel) return

    const message: TabMessage = {
      type,
      data,
      timestamp: Date.now(),
      tabId: this.tabId
    }

    this.channel.postMessage(message)
  }

  // SSE Connection Management
  public createSSEConnection(deviceId: string, url: string): EventSource | null {
    if (!this.isPrimary) {
      return null
    }

    // Close existing connection for this device
    const existing = this.sseConnections.get(deviceId)
    if (existing) {
      existing.close()
    }

    try {
      const eventSource = new EventSource(url, { withCredentials: true })
      
      eventSource.onmessage = (event) => {
        try {
          const parsedEvent = JSON.parse(event.data)
          
          if (parsedEvent.type === 'device_status_updated') {
            const data = parsedEvent.data
            
            // Broadcast to other tabs
            this.broadcast('device_update', data)
            
            // Handle locally
            this.handleDeviceUpdate(data)
          }
        } catch (parseError) {
          console.warn('Failed to parse SSE event:', parseError)
        }
      }

      eventSource.onerror = (error) => {
        console.warn(`SSE connection error for device ${deviceId}:`, error)
        this.sseConnections.delete(deviceId)
      }

      this.sseConnections.set(deviceId, eventSource)

      return eventSource
    } catch (error) {
      console.error(`Failed to create SSE connection for device ${deviceId}:`, error)
      return null
    }
  }

  public closeSSEConnection(deviceId: string) {
    const connection = this.sseConnections.get(deviceId)
    if (connection) {
      connection.close()
      this.sseConnections.delete(deviceId)
    }
  }

  public closeAllSSEConnections() {
    this.sseConnections.forEach((connection, deviceId) => {
      connection.close()
    })
    this.sseConnections.clear()
  }

  private cleanup() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval)
    }
    
    this.closeAllSSEConnections()
    
    if (this.channel) {
      this.channel.close()
    }
  }
}

// Singleton instance
let tabManager: TabManager | null = null

export function getTabManager(): TabManager {
  if (!tabManager) {
    tabManager = new TabManager()
  }
  return tabManager
}

export type { DeviceUpdate }