import { useState } from "react";
import { useAuth } from "@/components/AuthProvider";
import { LoginForm } from "@/components/LoginForm";
import { DeviceManagement } from "@/components/DeviceManagement";
import { PluginManagement } from "@/components/PluginManagement";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Monitor, Puzzle, PlayCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

export default function HomePage() {
  const { isAuthenticated, isLoading, login, authConfigured } = useAuth();
  const { t } = useTranslation();
  
  // Dialog states
  const [showDeviceManagement, setShowDeviceManagement] = useState(false);
  const [showPluginManagement, setShowPluginManagement] = useState(false);

  if (isLoading) {
    return null;
  }

  if (authConfigured && !isAuthenticated) {
    return <LoginForm onLogin={login} />;
  }

  return (
    <div className="bg-background pt-0 pb-8 px-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-2xl">Welcome to TRMNL Stationmaster</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-6">
              Your self-hosted solution for managing TRMNL devices, plugins, and content playlists.
            </p>
            
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => setShowDeviceManagement(true)}>
                <CardContent className="p-6 text-center">
                  <Monitor className="h-12 w-12 mx-auto mb-4 text-primary" />
                  <h3 className="font-semibold mb-2">Devices</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Link and manage your TRMNL devices
                  </p>
                  <Button variant="outline" className="w-full">
                    Manage Devices
                  </Button>
                </CardContent>
              </Card>

              <Card className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => setShowPluginManagement(true)}>
                <CardContent className="p-6 text-center">
                  <Puzzle className="h-12 w-12 mx-auto mb-4 text-primary" />
                  <h3 className="font-semibold mb-2">Plugins</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Configure content plugins and instances
                  </p>
                  <Button variant="outline" className="w-full">
                    Manage Plugins
                  </Button>
                </CardContent>
              </Card>

              <Card className="cursor-pointer hover:shadow-md transition-shadow opacity-50">
                <CardContent className="p-6 text-center">
                  <PlayCircle className="h-12 w-12 mx-auto mb-4 text-primary" />
                  <h3 className="font-semibold mb-2">Playlists</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Create and schedule content playlists
                  </p>
                  <Button variant="outline" className="w-full" disabled>
                    Coming Soon
                  </Button>
                </CardContent>
              </Card>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle>Quick Start Guide</CardTitle>
          </CardHeader>
          <CardContent>
            <ol className="list-decimal list-inside space-y-2 text-sm">
              <li>Link your TRMNL device using the device management interface</li>
              <li>Create plugin instances with your desired settings</li>
              <li>Build playlists to organize your content</li>
              <li>Schedule when different content should be displayed</li>
            </ol>
          </CardContent>
        </Card>
      </div>

      {/* Device Management Dialog */}
      <DeviceManagement
        isOpen={showDeviceManagement}
        onClose={() => setShowDeviceManagement(false)}
      />

      {/* Plugin Management Dialog */}
      <PluginManagement
        isOpen={showPluginManagement}
        onClose={() => setShowPluginManagement(false)}
      />
    </div>
  );
}