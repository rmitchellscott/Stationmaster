import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PageCard, PageCardContent, PageCardHeader, PageCardTitle } from "@/components/ui/page-card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { 
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
} from "@/components/ui/command";
import {
  ArrowLeft,
  Puzzle,
  Search,
} from "lucide-react";

interface Plugin {
  id: string;
  name: string;
  type: string;
  description: string;
  author: string;
}

type PluginType = 'all' | 'system' | 'private';

export function AddPluginPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [filteredPlugins, setFilteredPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedType, setSelectedType] = useState<PluginType>('all');
  const [expandedPlugin, setExpandedPlugin] = useState<Plugin | null>(null);

  // Fetch plugins
  const fetchPlugins = async () => {
    try {
      setLoading(true);
      const response = await fetch("/api/plugin-definitions", {
        credentials: "include",
      });
      if (response.ok) {
        const data = await response.json();
        setPlugins(data.plugins || []);
      }
    } catch (error) {
      console.error("Failed to fetch plugins:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPlugins();
  }, []);

  // Filter and search plugins
  useEffect(() => {
    let filtered = plugins;

    // Filter by type
    if (selectedType !== 'all') {
      filtered = filtered.filter(plugin => plugin.type === selectedType);
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(plugin => 
        plugin.name.toLowerCase().includes(query) ||
        plugin.description.toLowerCase().includes(query) ||
        plugin.author.toLowerCase().includes(query)
      );
    }

    setFilteredPlugins(filtered);
  }, [plugins, searchQuery, selectedType]);

  const handlePluginSelect = (plugin: Plugin) => {
    // Navigate back to plugin management with selected plugin info
    navigate('/?tab=plugins&subtab=instances&action=create&pluginId=' + plugin.id);
  };

  const handleCardClick = (plugin: Plugin) => {
    setExpandedPlugin(plugin);
  };

  const handleCloseExpanded = () => {
    setExpandedPlugin(null);
  };

  // Handle escape key to close expanded view
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && expandedPlugin) {
        handleCloseExpanded();
      }
    };

    if (expandedPlugin) {
      document.addEventListener('keydown', handleKeyDown);
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = 'unset';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [expandedPlugin]);

  const getPluginTypeBadge = (type: string) => {
    if (type === 'system') {
      return <Badge variant="outline">Native</Badge>;
    }
    return <Badge variant="outline">Private</Badge>;
  };

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4 py-4 space-y-4">
          {/* Breadcrumb */}
          <div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/?tab=plugins&subtab=instances')}
              className="gap-2 text-muted-foreground hover:text-foreground"
            >
              <ArrowLeft className="h-3 w-3" />
              Back to Plugin Management
            </Button>
          </div>
          
          {/* Title and Subtitle */}
          <div>
            <h1 className="text-2xl font-semibold">Add Plugin Instance</h1>
            <p className="text-muted-foreground">Choose a plugin to create an instance</p>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        <div className="container mx-auto px-4 py-6 space-y-6">
          {/* Search and Filter Controls */}
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search plugins by name, description, or author..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <div className="flex gap-2">
              <Button
                variant={selectedType === 'all' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedType('all')}
                className="w-[60px]"
              >
                All
              </Button>
              <Button
                variant={selectedType === 'system' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedType('system')}
                className="w-[70px]"
              >
                Native
              </Button>
              <Button
                variant={selectedType === 'private' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedType('private')}
                className="w-[70px]"
              >
                Private
              </Button>
            </div>
          </div>

          {/* Plugin Grid */}
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-muted-foreground">Loading plugins...</div>
            </div>
          ) : filteredPlugins.length === 0 ? (
            <Card className="py-12">
              <CardContent className="text-center">
                <Puzzle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">
                  {searchQuery || selectedType !== 'all' ? 'No matching plugins' : 'No plugins available'}
                </h3>
                <p className="text-muted-foreground">
                  {searchQuery || selectedType !== 'all' 
                    ? 'Try adjusting your search terms or filters'
                    : 'No plugins have been installed yet'
                  }
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
              {filteredPlugins.map((plugin) => (
                <Card 
                  key={plugin.id} 
                  className="cursor-pointer hover:shadow-md transition-all duration-200 hover:scale-[1.02]"
                  onClick={() => handleCardClick(plugin)}
                >
                  <CardContent className="px-3">
                    <div className="space-y-1">
                      {/* Line 1: Title + Badge on same line */}
                      <div className="flex items-start justify-between gap-2">
                        <h3 className="text-base font-semibold leading-tight flex-1 min-w-0">
                          {plugin.name}
                        </h3>
                        {getPluginTypeBadge(plugin.type)}
                      </div>
                      
                      {/* Line 2: Author */}
                      <p className="text-sm text-muted-foreground">
                        by {plugin.author}
                      </p>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Expanded Plugin Modal */}
      {expandedPlugin && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          {/* Backdrop */}
          <div 
            className="absolute inset-0 bg-background/80 backdrop-blur-sm"
            onClick={handleCloseExpanded}
          />
          
          {/* Expanded Card */}
          <Card className="relative z-10 w-full max-w-lg mx-auto animate-in fade-in-0 zoom-in-95 duration-300">
            <CardHeader className="pb-4">
              <div className="flex items-start justify-between gap-4">
                <CardTitle className="text-xl font-semibold">
                  {expandedPlugin.name}
                </CardTitle>
                {getPluginTypeBadge(expandedPlugin.type)}
              </div>
              <div className="space-y-1">
                <p className="text-sm text-muted-foreground">
                  by {expandedPlugin.author}
                </p>
                <p className="text-xs text-muted-foreground">
                  Version {expandedPlugin.version}
                </p>
              </div>
            </CardHeader>
            
            <CardContent className="space-y-6">
              <div>
                <div 
                  className="text-sm text-muted-foreground leading-relaxed"
                  dangerouslySetInnerHTML={{
                    __html: expandedPlugin.description || "No description available."
                  }}
                />
              </div>
              
              <div className="flex gap-3">
                <Button
                  variant="outline"
                  onClick={handleCloseExpanded}
                  className="flex-1"
                >
                  Cancel
                </Button>
                <Button
                  onClick={() => handlePluginSelect(expandedPlugin)}
                  className="flex-1"
                >
                  Create Instance
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}