package database

import (
	"github.com/google/uuid"
)

// ClearMirroredPlaylistsForSourceDevice removes playlist items from all devices mirroring the given source device
func (pls *PlaylistService) ClearMirroredPlaylistsForSourceDevice(sourceDeviceID uuid.UUID) error {
	// Find all devices that are mirroring this source device
	var mirroredDevices []Device
	if err := pls.db.Where("mirror_source_id = ?", sourceDeviceID).Find(&mirroredDevices).Error; err != nil {
		return err
	}

	// Clear playlist items for each mirrored device and remove the mirror relationship
	for _, device := range mirroredDevices {
		// Clear the mirrored playlists
		if err := pls.ClearMirroredPlaylists(device.ID); err != nil {
			return err
		}
		
		// Clear the mirror relationship
		device.MirrorSourceID = nil
		device.MirrorSyncedAt = nil
		if err := pls.db.Save(&device).Error; err != nil {
			return err
		}
	}

	return nil
}