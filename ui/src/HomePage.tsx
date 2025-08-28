import { useState, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useAuth } from "@/components/AuthProvider";
import { LoginForm } from "@/components/LoginForm";
import { PluginManagement } from "@/components/PluginManagement";
import { PlaylistManagement } from "@/components/PlaylistManagement";
import { DeviceSelector } from "@/components/DeviceSelector";
import { DeviceManagementContent } from "@/components/DeviceManagementContent";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PageCard, PageCardContent, PageCardHeader, PageCardTitle } from "@/components/ui/page-card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Device } from "@/utils/deviceHelpers";
import { Puzzle, PlayCircle, Monitor } from "lucide-react";
import { useTranslation } from "react-i18next";

const SELECTED_DEVICE_KEY = "stationmaster_selected_device";

const getStoredDeviceId = (): string | null => {
  try {
    return localStorage.getItem(SELECTED_DEVICE_KEY);
  } catch {
    return null;
  }
};

const storeDeviceId = (deviceId: string | null) => {
  try {
    if (deviceId) {
      localStorage.setItem(SELECTED_DEVICE_KEY, deviceId);
    } else {
      localStorage.removeItem(SELECTED_DEVICE_KEY);
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

  // Real-time device updates
  useEffect(() => {
    if (!isAuthenticated || devices.length === 0) return;

    const eventSources: EventSource[] = [];
    
    // Create SSE connections for each device
    devices.forEach(device => {
      try {
        const eventSource = new EventSource(`/api/devices/${device.id}/events`, {
          withCredentials: true,
        });

        eventSource.onmessage = (event) => {
          try {
            const parsedEvent = JSON.parse(event.data);
            
            if (parsedEvent.type === 'device_status_updated') {
              const data = parsedEvent.data;
              
              // Update the specific device in the devices array
              setDevices(prevDevices => 
                prevDevices.map(d => 
                  d.id === data.device_id 
                    ? {
                        ...d,
                        battery_voltage: data.battery_voltage,
                        rssi: data.rssi,
                        firmware_version: data.firmware_version,
                        last_seen: data.last_seen,
                        is_active: data.is_active,
                      }
                    : d
                )
              );
            }
          } catch (parseError) {
          }
        };

        eventSource.onerror = (error) => {
        };

        eventSources.push(eventSource);
      } catch (error) {
      }
    });

    // Cleanup function
    return () => {
      eventSources.forEach(eventSource => {
        eventSource.close();
      });
    };
  }, [isAuthenticated, devices.length]); // Only depend on devices.length to avoid infinite loops

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
    <div className="bg-background pt-0 pb-8 px-0 sm:px-8">
      <div className="max-w-6xl mx-0 sm:mx-auto space-y-6">
        {showOnboarding && (
          <PageCard>
            <PageCardHeader>
              <PageCardTitle>Welcome to Stationmaster</PageCardTitle>
            </PageCardHeader>
            <PageCardContent>
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
            </PageCardContent>
          </PageCard>
        )}
        
        <PageCard>
          <PageCardHeader>
            <PageCardTitle className="text-2xl">Dashboard</PageCardTitle>
          </PageCardHeader>
          <PageCardContent>
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
                    storeDeviceId(deviceId);
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
          </PageCardContent>
        </PageCard>
      </div>
    </div>
  );
}
