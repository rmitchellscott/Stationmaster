import { useState, useEffect, useCallback, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useAuth } from "@/components/AuthProvider";
import { LoginForm } from "@/components/LoginForm";
import { PluginManagement } from "@/components/PluginManagement";
import { PlaylistManagement } from "@/components/PlaylistManagement";
import { DeviceSelector } from "@/components/DeviceSelector";
import { DeviceManagementContent } from "@/components/DeviceManagementContent";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Device } from "@/utils/deviceHelpers";
import { Puzzle, PlayCircle, Monitor } from "lucide-react";
import { useTranslation } from "react-i18next";
import { getTabManager, DeviceUpdate } from "@/utils/tabManager";

const SELECTED_DEVICE_KEY = "stationmaster_selected_device";
const SELECTED_DEVICE_SESSION_KEY = "stationmaster_selected_device_session";

const getStoredDeviceId = (): string | null => {
  try {
    // Check session storage first (tab-specific)
    const sessionDeviceId = sessionStorage.getItem(SELECTED_DEVICE_SESSION_KEY)
    if (sessionDeviceId) {
      return sessionDeviceId
    }
    // Fall back to localStorage (global)
    return localStorage.getItem(SELECTED_DEVICE_KEY);
  } catch {
    return null;
  }
};

const storeDeviceId = (deviceId: string | null, persistent: boolean = true) => {
  try {
    if (deviceId) {
      // Always store in session for current tab
      sessionStorage.setItem(SELECTED_DEVICE_SESSION_KEY, deviceId)
      // Also store in localStorage for persistence across sessions if requested
      if (persistent) {
        localStorage.setItem(SELECTED_DEVICE_KEY, deviceId);
      }
    } else {
      sessionStorage.removeItem(SELECTED_DEVICE_SESSION_KEY)
      if (persistent) {
        localStorage.removeItem(SELECTED_DEVICE_KEY);
      }
    }
  } catch (error) {
    console.warn("Failed to store device ID:", error);
  }
};

export default function HomePage() {
  const { isAuthenticated, isLoading, login, authConfigured, user } = useAuth();
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  
  // State
  const [devices, setDevices] = useState<Device[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [pluginInstances, setPluginInstances] = useState([]);
  const [pluginInstancesLoading, setPluginInstancesLoading] = useState(false);
  const [playlistItems, setPlaylistItems] = useState([]);
  const [playlistItemsLoading, setPlaylistItemsLoading] = useState(false);
  const [showOnboarding, setShowOnboarding] = useState(true);
  
  // Tab management
  const tabManager = getTabManager()
  const [isPrimaryTab, setIsPrimaryTab] = useState(tabManager.isPrimaryTab())

  // Get active tab from URL query parameters
  const activeTab = searchParams.get('tab') || 'plugins';

  // Handle tab change by updating URL query parameters
  const handleTabChange = (tab: string) => {
    const newSearchParams = new URLSearchParams(searchParams);
    newSearchParams.set('tab', tab);
    // Remove subtab when changing main tab to avoid confusion
    newSearchParams.delete('subtab');
    setSearchParams(newSearchParams);
  };

  // Fetch devices
  const fetchDevices = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/devices", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        const fetchedDevices = data.devices || [];
        setDevices(fetchedDevices);
        
        // Handle device selection persistence
        const storedDeviceId = getStoredDeviceId();
        const storedDeviceExists = fetchedDevices.some((d: Device) => d.id === storedDeviceId);
        const currentlySelectedExists = selectedDeviceId ? fetchedDevices.some((d: Device) => d.id === selectedDeviceId) : false;
        
        // Check if currently selected device still exists (handles unlinking during session)
        if (selectedDeviceId && !currentlySelectedExists) {
          // Currently selected device was unlinked, need to select a new one
          if (fetchedDevices.length > 0) {
            const firstDeviceId = fetchedDevices[0].id;
            setSelectedDeviceId(firstDeviceId);
            storeDeviceId(firstDeviceId);
          } else {
            // No devices left
            setSelectedDeviceId(null);
            storeDeviceId(null);
          }
        } else if (storedDeviceId && storedDeviceExists) {
          // Use stored device if it still exists
          setSelectedDeviceId(storedDeviceId);
        } else if (!selectedDeviceId && fetchedDevices.length > 0) {
          // Auto-select first device if none selected and no valid stored device
          const firstDeviceId = fetchedDevices[0].id;
          setSelectedDeviceId(firstDeviceId);
          storeDeviceId(firstDeviceId);
        } else if (storedDeviceId && !storedDeviceExists) {
          // Clear stored device if it no longer exists
          storeDeviceId(null);
          if (fetchedDevices.length > 0) {
            const firstDeviceId = fetchedDevices[0].id;
            setSelectedDeviceId(firstDeviceId);
            storeDeviceId(firstDeviceId);
          } else {
            // No devices available
            setSelectedDeviceId(null);
          }
        }
      }
    } catch (error) {
      console.error("Failed to fetch devices:", error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch user plugins
  const fetchPluginInstances = async () => {
    try {
      setPluginInstancesLoading(true);
      const response = await fetch("/api/plugin-instances", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPluginInstances(data.plugin_instances || []);
      }
    } catch (error) {
      console.error("Failed to fetch plugin instances:", error);
    } finally {
      setPluginInstancesLoading(false);
    }
  };

  // Fetch playlist items for selected device
  const fetchPlaylistItems = async () => {
    if (!selectedDeviceId) {
      setPlaylistItems([]);
      setPlaylistItemsLoading(false);
      return;
    }
    
    try {
      setPlaylistItemsLoading(true);
      // Get the default playlist for this device
      const playlistsResponse = await fetch(`/api/playlists?device_id=${selectedDeviceId}`, {
        credentials: "include",
      });
      
      if (playlistsResponse.ok) {
        const playlistsText = await playlistsResponse.text();
        try {
          const playlistsData = JSON.parse(playlistsText);
          const defaultPlaylist = playlistsData.playlists?.find((p: any) => p.is_default);
          
          if (defaultPlaylist) {
            const playlistDetailsResponse = await fetch(`/api/playlists/${defaultPlaylist.id}`, {
              credentials: "include",
            });
            if (playlistDetailsResponse.ok) {
              const playlistDetailsText = await playlistDetailsResponse.text();
              try {
                const playlistDetailsData = JSON.parse(playlistDetailsText);
                setPlaylistItems(playlistDetailsData.items || []);
              } catch (parseError) {
                console.error("Failed to parse playlist details JSON:", parseError);
                console.error("Response text:", playlistDetailsText);
                setPlaylistItems([]);
              }
            } else {
              const errorText = await playlistDetailsResponse.text();
              console.error("Failed to fetch playlist details - HTTP", playlistDetailsResponse.status);
              console.error("Error response:", errorText);
              
              // Check if it's an authentication issue
              if (playlistDetailsResponse.status === 401) {
                console.error("Authentication error - user may need to log in again");
              } else if (playlistDetailsResponse.status === 403) {
                console.error("Permission denied - user may not have access to this playlist");
              } else if (errorText.includes("DOCTYPE")) {
                console.error("Server returned HTML instead of JSON - possible routing or middleware issue");
              }
              
              setPlaylistItems([]);
            }
          } else {
            console.log("No default playlist found for device, setting empty playlist items");
            setPlaylistItems([]);
          }
        } catch (parseError) {
          console.error("Failed to parse playlists JSON:", parseError);
          console.error("Response text:", playlistsText);
          setPlaylistItems([]);
        }
      } else {
        const errorText = await playlistsResponse.text();
        console.error("Failed to fetch playlists - HTTP", playlistsResponse.status);
        console.error("Error response:", errorText);
        
        // Check if it's an authentication issue
        if (playlistsResponse.status === 401) {
          console.error("Authentication error - user may need to log in again");
        } else if (playlistsResponse.status === 403) {
          console.error("Permission denied - user may not have access to this device");
        } else if (errorText.includes("DOCTYPE")) {
          console.error("Server returned HTML instead of JSON - possible routing or middleware issue");
        }
        
        setPlaylistItems([]);
      }
    } catch (error) {
      console.error("Failed to fetch playlist items:", error);
      setPlaylistItems([]);
    } finally {
      setPlaylistItemsLoading(false);
    }
  };

  // Complete onboarding and update state
  const completeOnboarding = async () => {
    try {
      const response = await fetch("/api/user/complete-onboarding", {
        method: "POST",
        credentials: "include",
      });
      if (response.ok) {
        setShowOnboarding(false);
      } else {
        console.error("Failed to complete onboarding");
      }
    } catch (error) {
      console.error("Failed to complete onboarding:", error);
    }
  };

  useEffect(() => {
    if (isAuthenticated) {
      fetchDevices();
      fetchPluginInstances();
    }
  }, [isAuthenticated]);

  // Handle device updates from tab manager
  const handleDeviceUpdate = useCallback((data: DeviceUpdate) => {
    setDevices(prevDevices => 
      prevDevices.map(d => 
        d.id === data.device_id 
          ? {
              ...d,
              battery_voltage: data.battery_voltage ?? d.battery_voltage,
              rssi: data.rssi ?? d.rssi,
              firmware_version: data.firmware_version ?? d.firmware_version,
              last_seen: data.last_seen ?? d.last_seen,
              is_active: data.is_active ?? d.is_active,
            }
          : d
      )
    );
  }, [])

  // Setup tab management and SSE connections
  useEffect(() => {
    if (!isAuthenticated) return;

    // Listen for primary tab changes
    const handlePrimaryChange = (isPrimary: boolean) => {
      setIsPrimaryTab(isPrimary)
    }

    // Listen for device updates from other tabs
    tabManager.addListener('device_update', handleDeviceUpdate)
    tabManager.addListener('primary_changed', handlePrimaryChange)

    return () => {
      tabManager.removeListener('device_update', handleDeviceUpdate)
      tabManager.removeListener('primary_changed', handlePrimaryChange)
    }
  }, [isAuthenticated, handleDeviceUpdate])

  // Create SSE connections only for primary tab
  useEffect(() => {
    if (!isAuthenticated || devices.length === 0 || !isPrimaryTab) {
      return;
    }

    console.log(`Primary tab creating SSE connections for ${devices.length} devices`)
    
    // Create SSE connections for each device via tab manager
    devices.forEach(device => {
      tabManager.createSSEConnection(device.id, `/api/devices/${device.id}/events`)
    })

    // Cleanup function
    return () => {
      devices.forEach(device => {
        tabManager.closeSSEConnection(device.id)
      })
    };
  }, [isAuthenticated, devices.length, isPrimaryTab]); // Include isPrimaryTab in dependencies

  useEffect(() => {
    if (selectedDeviceId) {
      fetchPlaylistItems();
    } else {
      setPlaylistItems([]);
    }
  }, [selectedDeviceId]);

  useEffect(() => {
    // Set onboarding status based on user data from auth context
    if (user) {
      setShowOnboarding(!user.onboarding_completed);
    }
  }, [user]);

  if (isLoading) {
    return null;
  }

  if (authConfigured && !isAuthenticated) {
    return <LoginForm onLogin={login} />;
  }

  return (
    <div className="min-h-screen">
      {/* Sticky Header */}
      <div className="sticky top-0 z-40 border-b bg-background">
        <div className="container mx-auto px-4 py-4 space-y-4">
          {/* Title and Subtitle */}
          <div>
            <h1 className="text-2xl font-semibold">Dashboard</h1>
            <p className="text-muted-foreground">Manage your TRMNL devices, plugins, and playlists</p>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="container mx-auto px-4 py-6 space-y-6">
        {showOnboarding && (
          <Card>
            <CardHeader>
              <CardTitle>Welcome to Stationmaster</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground mb-4">
                Your self-hosted solution for managing TRMNL devices.
              </p>
              <div className="bg-muted/50 rounded-lg p-4">
                <h4 className="font-medium mb-2">Quick Start Guide</h4>
                <ol className="list-decimal list-inside space-y-1 text-sm text-muted-foreground">
                  <li>Add a device in the Devices tab</li>
                  <li>Create plugin instances in the Plugins tab</li>
                  <li>Select a device and add playlist items in the Playlist tab</li>
                </ol>
              </div>
              <div className="flex justify-end mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={completeOnboarding}
                >
                  Don't show again
                </Button>
              </div>
            </CardContent>
          </Card>
        )}
            <Tabs value={activeTab} onValueChange={handleTabChange} className="w-full">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="plugins">
                  <Puzzle className="h-4 w-4 mr-2" />
                  Plugins
                </TabsTrigger>
                <TabsTrigger value="playlist">
                  <PlayCircle className="h-4 w-4 mr-2" />
                  Playlist
                </TabsTrigger>
                <TabsTrigger value="devices">
                  <Monitor className="h-4 w-4 mr-2" />
                  Devices
                </TabsTrigger>
              </TabsList>
              
              <TabsContent value="plugins" className="mt-6">
                <PluginManagement 
                  selectedDeviceId={selectedDeviceId || ""}
                  onUpdate={fetchPluginInstances}
                />
              </TabsContent>
              
              <TabsContent value="playlist" className="mt-6 space-y-6">
                <DeviceSelector
                  devices={devices}
                  selectedDeviceId={selectedDeviceId}
                  onDeviceChange={(deviceId) => {
                    setSelectedDeviceId(deviceId);
                    storeDeviceId(deviceId, true);
                  }}
                  loading={loading}
                />
                {selectedDeviceId && (
                  <PlaylistManagement 
                    selectedDeviceId={selectedDeviceId}
                    devices={devices}
                    onUpdate={fetchPlaylistItems}
                  />
                )}
              </TabsContent>
              
              <TabsContent value="devices" className="mt-6">
                <DeviceManagementContent onUpdate={fetchDevices} />
              </TabsContent>
            </Tabs>
      </div>
    </div>
  );
}
