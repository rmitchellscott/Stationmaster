// Mashup API service for frontend
export interface MashupLayout {
  id: string;
  name: string;
  description: string;
  slots: number;
  icon: string;
}

export interface MashupSlotInfo {
  position: string;
  view_class: string;
  display_name: string;
  required_size: string;
}

export interface MashupChild {
  instance_id: string;
  instance_name: string;
  plugin_name: string;
  plugin_type: string;
}

export interface MashupChildrenResponse {
  layout: string;
  slots: MashupSlotInfo[];
  assignments: Record<string, MashupChild>;
}

export interface AvailablePluginInstance {
  id: string;
  name: string;
  plugin_name: string;
  plugin_description: string;
  refresh_interval: number;
}

export interface CreateMashupRequest {
  name: string;
  description?: string;
  layout: string;
}

export interface CreateMashupResponse {
  mashup: {
    id: string;
    name: string;
    description: string;
    layout: string;
    slots: MashupSlotInfo[];
  };
}

export interface AssignChildrenRequest {
  assignments: Record<string, string>; // slot -> instance_id
}

class MashupService {
  private baseURL = "/api";

  private async fetchWithCredentials(url: string, options: RequestInit = {}): Promise<Response> {
    return fetch(url, {
      credentials: "include",
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
    });
  }

  // Create new mashup definition
  async createMashup(request: CreateMashupRequest): Promise<CreateMashupResponse> {
    const response = await this.fetchWithCredentials(`${this.baseURL}/plugin-definitions/mashup`, {
      method: "POST",
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || "Failed to create mashup");
    }

    return response.json();
  }

  // Get available mashup layouts
  async getAvailableLayouts(): Promise<MashupLayout[]> {
    const response = await this.fetchWithCredentials(`${this.baseURL}/plugin-definitions/mashup/layouts`);

    if (!response.ok) {
      throw new Error("Failed to fetch available layouts");
    }

    const data = await response.json();
    return data.layouts;
  }

  // Get slot configuration for a specific layout
  async getLayoutSlots(layout: string): Promise<{ layout: string; slots: MashupSlotInfo[] }> {
    const response = await this.fetchWithCredentials(`${this.baseURL}/plugin-definitions/mashup/layouts/${layout}/slots`);

    if (!response.ok) {
      throw new Error(`Failed to fetch slots for layout ${layout}`);
    }

    return response.json();
  }

  // Assign child plugin instances to mashup slots
  async assignChildren(instanceId: string, assignments: Record<string, string>): Promise<void> {
    const response = await this.fetchWithCredentials(`${this.baseURL}/plugin-instances/${instanceId}/mashup/children`, {
      method: "POST",
      body: JSON.stringify({ assignments }),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || "Failed to assign children to mashup");
    }
  }

  // Get current child assignments for a mashup
  async getChildren(instanceId: string): Promise<MashupChildrenResponse> {
    const response = await this.fetchWithCredentials(`${this.baseURL}/plugin-instances/${instanceId}/mashup/children`);

    if (!response.ok) {
      throw new Error("Failed to get mashup children");
    }

    return response.json();
  }

  // Get user's private plugin instances available for mashup children
  async getAvailablePluginInstances(): Promise<AvailablePluginInstance[]> {
    const response = await this.fetchWithCredentials(`${this.baseURL}/plugin-instances/private`);

    if (!response.ok) {
      throw new Error("Failed to get available plugin instances");
    }

    const data = await response.json();
    return data.instances;
  }
}

// Export singleton instance
export const mashupService = new MashupService();