// Mashup API service functions

export interface MashupLayout {
  id: string;
  name: string;
  description: string;
  positions: string[];
}

export interface CreateMashupRequest {
  name: string;
  description: string;
  mashup_layout: string;
}

export interface AddMashupChildRequest {
  child_instance_id: string;
  grid_position: string;
}

export interface UpdateMashupChildPositionRequest {
  grid_position: string;
}

export interface MashupChild {
  id: string;
  mashup_instance_id: string;
  child_instance_id: string;
  grid_position: string;
  child_instance: {
    id: string;
    name: string;
    plugin_definition?: {
      name: string;
      type: string;
    };
    refresh_interval: number;
    is_active: boolean;
  };
}

export class MashupApiService {
  private static baseUrl = '/api';

  // Create a new mashup definition
  static async createMashupDefinition(data: CreateMashupRequest): Promise<{ id: string; message: string }> {
    const response = await fetch(`${this.baseUrl}/mashups`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create mashup definition');
    }

    return response.json();
  }

  // Get available mashup layouts
  static async getAvailableLayouts(): Promise<{ layouts: MashupLayout[] }> {
    const response = await fetch(`${this.baseUrl}/mashups/layouts`, {
      credentials: 'include',
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch mashup layouts');
    }

    return response.json();
  }

  // Add a child plugin to a mashup instance
  static async addMashupChild(
    mashupInstanceId: string, 
    data: AddMashupChildRequest
  ): Promise<{ message: string }> {
    const response = await fetch(`${this.baseUrl}/plugin-instances/${mashupInstanceId}/children`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to add child plugin');
    }

    return response.json();
  }

  // Remove a child plugin from a mashup instance
  static async removeMashupChild(
    mashupInstanceId: string, 
    childInstanceId: string
  ): Promise<{ message: string }> {
    const response = await fetch(
      `${this.baseUrl}/plugin-instances/${mashupInstanceId}/children/${childInstanceId}`,
      {
        method: 'DELETE',
        credentials: 'include',
      }
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to remove child plugin');
    }

    return response.json();
  }

  // Update a child plugin's position in a mashup
  static async updateMashupChildPosition(
    mashupInstanceId: string,
    childInstanceId: string,
    data: UpdateMashupChildPositionRequest
  ): Promise<{ message: string }> {
    const response = await fetch(
      `${this.baseUrl}/plugin-instances/${mashupInstanceId}/children/${childInstanceId}/position`,
      {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(data),
      }
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update child plugin position');
    }

    return response.json();
  }

  // Get all child plugins for a mashup instance
  static async getMashupChildren(mashupInstanceId: string): Promise<{ children: MashupChild[] }> {
    const response = await fetch(
      `${this.baseUrl}/plugin-instances/${mashupInstanceId}/children`,
      {
        credentials: 'include',
      }
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch mashup children');
    }

    return response.json();
  }
}

export default MashupApiService;