import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Alert,
  AlertDescription,
} from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Grid2x2,
  ColumnsIcon,
  Rows3Icon,
  Rows2Icon,
  Plus,
  Edit,
  Trash2,
  Search,
  AlertTriangle,
  RefreshCw,
  Settings,
} from "lucide-react";

interface MashupDefinition {
  id: string;
  name: string;
  description: string;
  type: string;
  plugin_type: string;
  mashup_layout?: string;
  instance_count?: number;
  created_at: string;
  updated_at: string;
  is_active: boolean;
}

interface MashupListProps {
  onCreateMashup: () => void;
  refreshTrigger?: number; // External refresh trigger
}

const getLayoutIcon = (layout?: string) => {
  switch (layout) {
    case "1L1R":
      return <ColumnsIcon className="h-4 w-4" />;
    case "2T1B":
      return <Rows2Icon className="h-4 w-4" />;
    case "1T2B": 
      return <Rows3Icon className="h-4 w-4" />;
    case "2x2":
      return <Grid2x2 className="h-4 w-4" />;
    default:
      return <Grid2x2 className="h-4 w-4" />;
  }
};

const getLayoutName = (layout?: string) => {
  switch (layout) {
    case "1L1R":
      return "Left & Right";
    case "2T1B":
      return "Two Top, One Bottom";
    case "1T2B":
      return "One Top, Two Bottom"; 
    case "2x2":
      return "Four Quadrants";
    default:
      return "Unknown Layout";
  }
};

export const MashupList: React.FC<MashupListProps> = ({
  onCreateMashup,
  refreshTrigger = 0,
}) => {
  const { t } = useTranslation();
  const [mashups, setMashups] = useState<MashupDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [deletingMashup, setDeletingMashup] = useState<MashupDefinition | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const fetchMashups = async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Fetch mashup plugin definitions specifically
      const response = await fetch("/api/plugin-definitions?plugin_type=mashup", {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setMashups(data.plugins || []);
      } else {
        setError("Failed to fetch mashups");
      }
    } catch (error) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMashups();
  }, [refreshTrigger]);

  const filteredMashups = mashups.filter((mashup) =>
    mashup.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    mashup.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleDeleteMashup = async (mashup: MashupDefinition) => {
    if (!mashup.id) return;
    
    setDeletingId(mashup.id);
    
    try {
      const response = await fetch(`/api/plugin-definitions/${mashup.id}`, {
        method: "DELETE",
        credentials: "include",
      });

      if (response.ok) {
        setMashups(prev => prev.filter(m => m.id !== mashup.id));
        setDeletingMashup(null);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to delete mashup");
      }
    } catch (error) {
      setError("Network error occurred while deleting mashup");
    } finally {
      setDeletingId(null);
    }
  };

  const createMashupInstance = async (mashupId: string) => {
    try {
      const response = await fetch("/api/plugin-instances", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          plugin_definition_id: mashupId,
          name: `Mashup Instance ${Date.now()}`,
          settings: {},
          refresh_interval: 3600, // Will be updated when children are added
        }),
      });

      if (response.ok) {
        // Refresh the list to update instance counts
        fetchMashups();
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to create mashup instance");
      }
    } catch (error) {
      setError("Network error occurred while creating instance");
    }
  };

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <RefreshCw className="h-6 w-6 animate-spin mr-2" />
          <span>Loading mashups...</span>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Header and Controls */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex-1 max-w-md relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground h-4 w-4" />
          <Input
            placeholder="Search mashups..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>
        
        <Button onClick={onCreateMashup} className="shrink-0">
          <Plus className="mr-2 h-4 w-4" />
          Create Mashup
        </Button>
      </div>

      {/* Mashup List */}
      {filteredMashups.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <Grid2x2 className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">
              {searchTerm ? "No mashups found" : "No mashups yet"}
            </h3>
            <p className="text-muted-foreground mb-6 max-w-md">
              {searchTerm 
                ? "Try adjusting your search terms to find the mashup you're looking for."
                : "Create your first mashup to combine multiple private plugins in a customizable grid layout."
              }
            </p>
            {!searchTerm && (
              <Button onClick={onCreateMashup}>
                <Plus className="mr-2 h-4 w-4" />
                Create Your First Mashup
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Grid2x2 className="h-5 w-5" />
              Mashups ({filteredMashups.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Layout</TableHead>
                  <TableHead>Instances</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredMashups.map((mashup) => (
                  <TableRow key={mashup.id}>
                    <TableCell>
                      <div>
                        <div className="font-medium">{mashup.name}</div>
                        {mashup.description && (
                          <div className="text-sm text-muted-foreground">
                            {mashup.description}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getLayoutIcon(mashup.mashup_layout)}
                        <span className="text-sm">
                          {getLayoutName(mashup.mashup_layout)}
                        </span>
                      </div>
                    </TableCell>
                    
                    <TableCell>
                      <Badge variant="secondary">
                        {mashup.instance_count || 0} instance{(mashup.instance_count || 0) !== 1 ? 's' : ''}
                      </Badge>
                    </TableCell>
                    
                    <TableCell>
                      <Badge variant={mashup.is_active ? "default" : "secondary"}>
                        {mashup.is_active ? "Active" : "Inactive"}
                      </Badge>
                    </TableCell>
                    
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(mashup.created_at).toLocaleDateString()}
                    </TableCell>
                    
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => createMashupInstance(mashup.id)}
                            >
                              <Settings className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>
                            Create Instance
                          </TooltipContent>
                        </Tooltip>
                        
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => setDeletingMashup(mashup)}
                              disabled={deletingId === mashup.id}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>
                            Delete Mashup
                          </TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog 
        open={deletingMashup !== null} 
        onOpenChange={(open) => !open && setDeletingMashup(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Mashup</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{deletingMashup?.name}"? This action cannot be undone.
              {(deletingMashup?.instance_count || 0) > 0 && (
                <div className="mt-2 p-2 bg-destructive/10 text-destructive text-sm rounded">
                  ⚠️ This mashup has {deletingMashup?.instance_count} instance(s) that will also be deleted.
                </div>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deletingId === deletingMashup?.id}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingMashup && handleDeleteMashup(deletingMashup)}
              disabled={deletingId === deletingMashup?.id}
              className="bg-destructive hover:bg-destructive/90"
            >
              {deletingId === deletingMashup?.id ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};