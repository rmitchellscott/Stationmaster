package database

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MashupService handles database operations for mashup plugins
type MashupService struct {
	db *gorm.DB
}

// NewMashupService creates a new mashup service
func NewMashupService(db *gorm.DB) *MashupService {
	return &MashupService{db: db}
}

// MashupSlotInfo defines metadata for a mashup slot
type MashupSlotInfo struct {
	Position     string `json:"position"`      // Slot identifier like "left", "right", "q1", etc.
	ViewClass    string `json:"view_class"`    // CSS class like "view--half_vertical"
	DisplayName  string `json:"display_name"`  // User-friendly name like "Left Panel"
	RequiredSize string `json:"required_size"` // Size requirement: "half", "quarter", "full"
}

// CreateMashupDefinition creates a new mashup plugin definition
func (s *MashupService) CreateMashupDefinition(userID uuid.UUID, name string, layout string) (*PluginDefinition, error) {
	// Generate slot metadata based on layout
	slots, err := s.generateSlotMetadata(layout)
	if err != nil {
		return nil, fmt.Errorf("invalid layout %s: %w", layout, err)
	}
	
	slotsJSON, err := json.Marshal(slots)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slots: %w", err)
	}
	
	definition := &PluginDefinition{
		ID:                 uuid.New().String(),
		PluginType:         "mashup",
		OwnerID:            &userID,
		Identifier:         uuid.New().String(),
		Name:               name,
		Description:        fmt.Sprintf("Mashup with %s layout", layout),
		Version:            "1.0.0",
		Author:             "Mashup Creator",
		ConfigSchema:       "{}",
		RequiresProcessing: true,
		IsMashup:           true,
		MashupLayout:       &layout,
		MashupSlots:        slotsJSON,
		IsActive:           true,
	}
	
	if err := s.db.Create(definition).Error; err != nil {
		return nil, fmt.Errorf("failed to create mashup definition: %w", err)
	}
	
	return definition, nil
}

// AssignChildren assigns child plugin instances to mashup slots
func (s *MashupService) AssignChildren(mashupInstanceID uuid.UUID, assignments map[string]uuid.UUID) error {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Delete existing children
	if err := tx.Where("mashup_instance_id = ?", mashupInstanceID).Delete(&MashupChild{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete existing children: %w", err)
	}
	
	// Create new child assignments
	for slot, childID := range assignments {
		child := &MashupChild{
			MashupInstanceID: mashupInstanceID,
			ChildInstanceID:  childID,
			SlotPosition:     slot,
		}
		if err := tx.Create(child).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to assign child to slot %s: %w", slot, err)
		}
	}
	
	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// GetChildren retrieves all child instances for a mashup
func (s *MashupService) GetChildren(mashupInstanceID uuid.UUID) ([]MashupChild, error) {
	var children []MashupChild
	err := s.db.Where("mashup_instance_id = ?", mashupInstanceID).
		Preload("ChildInstance").
		Preload("ChildInstance.PluginDefinition").
		Find(&children).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get mashup children: %w", err)
	}
	return children, nil
}

// CalculateRefreshRate calculates the minimum refresh rate among all children
func (s *MashupService) CalculateRefreshRate(mashupInstanceID uuid.UUID) (int, error) {
	children, err := s.GetChildren(mashupInstanceID)
	if err != nil {
		return 0, err
	}
	
	if len(children) == 0 {
		// Default to hourly if no children
		return 3600, nil
	}
	
	// Find minimum refresh rate
	minRate := math.MaxInt32
	for _, child := range children {
		if child.ChildInstance.RefreshInterval < minRate {
			minRate = child.ChildInstance.RefreshInterval
		}
	}
	
	return minRate, nil
}

// ValidateMashupChild validates that a plugin instance can be used as a mashup child
func (s *MashupService) ValidateMashupChild(instanceID uuid.UUID) error {
	var instance PluginInstance
	err := s.db.Preload("PluginDefinition").First(&instance, instanceID).Error
	if err != nil {
		return fmt.Errorf("plugin instance not found: %w", err)
	}
	
	// Only private and external plugins can be mashup children
	if instance.PluginDefinition.PluginType != "private" && instance.PluginDefinition.PluginType != "external" {
		return fmt.Errorf("only private and external plugins can be used in mashups")
	}
	
	// Don't allow mashups as children (no nesting)
	if instance.PluginDefinition.IsMashup {
		return fmt.Errorf("mashups cannot contain other mashups")
	}
	
	return nil
}

// generateSlotMetadata generates slot configuration based on layout type
func (s *MashupService) generateSlotMetadata(layout string) ([]MashupSlotInfo, error) {
	switch layout {
	case "1Lx1R": // 1 Left, 1 Right
		return []MashupSlotInfo{
			{Position: "left", ViewClass: "view--half_vertical", DisplayName: "Left Panel", RequiredSize: "half"},
			{Position: "right", ViewClass: "view--half_vertical", DisplayName: "Right Panel", RequiredSize: "half"},
		}, nil
		
	case "1Tx1B": // 1 Top, 1 Bottom
		return []MashupSlotInfo{
			{Position: "top", ViewClass: "view--half_horizontal", DisplayName: "Top Panel", RequiredSize: "half"},
			{Position: "bottom", ViewClass: "view--half_horizontal", DisplayName: "Bottom Panel", RequiredSize: "half"},
		}, nil
		
	case "1Lx2R": // 1 Left, 2 Right
		return []MashupSlotInfo{
			{Position: "left", ViewClass: "view--half_vertical", DisplayName: "Left Panel", RequiredSize: "half"},
			{Position: "right-top", ViewClass: "view--quadrant", DisplayName: "Right Top", RequiredSize: "quarter"},
			{Position: "right-bottom", ViewClass: "view--quadrant", DisplayName: "Right Bottom", RequiredSize: "quarter"},
		}, nil
		
	case "2Lx1R": // 2 Left, 1 Right
		return []MashupSlotInfo{
			{Position: "left-top", ViewClass: "view--quadrant", DisplayName: "Left Top", RequiredSize: "quarter"},
			{Position: "left-bottom", ViewClass: "view--quadrant", DisplayName: "Left Bottom", RequiredSize: "quarter"},
			{Position: "right", ViewClass: "view--half_vertical", DisplayName: "Right Panel", RequiredSize: "half"},
		}, nil
		
	case "2Tx1B": // 2 Top, 1 Bottom
		return []MashupSlotInfo{
			{Position: "top-left", ViewClass: "view--quadrant", DisplayName: "Top Left", RequiredSize: "quarter"},
			{Position: "top-right", ViewClass: "view--quadrant", DisplayName: "Top Right", RequiredSize: "quarter"},
			{Position: "bottom", ViewClass: "view--half_horizontal", DisplayName: "Bottom Panel", RequiredSize: "half"},
		}, nil
		
	case "1Tx2B": // 1 Top, 2 Bottom
		return []MashupSlotInfo{
			{Position: "top", ViewClass: "view--half_horizontal", DisplayName: "Top Panel", RequiredSize: "half"},
			{Position: "bottom-left", ViewClass: "view--quadrant", DisplayName: "Bottom Left", RequiredSize: "quarter"},
			{Position: "bottom-right", ViewClass: "view--quadrant", DisplayName: "Bottom Right", RequiredSize: "quarter"},
		}, nil
		
	case "2x2": // 2x2 Grid (4 quadrants)
		return []MashupSlotInfo{
			{Position: "q1", ViewClass: "view--quadrant", DisplayName: "Top Left", RequiredSize: "quarter"},
			{Position: "q2", ViewClass: "view--quadrant", DisplayName: "Top Right", RequiredSize: "quarter"},
			{Position: "q3", ViewClass: "view--quadrant", DisplayName: "Bottom Left", RequiredSize: "quarter"},
			{Position: "q4", ViewClass: "view--quadrant", DisplayName: "Bottom Right", RequiredSize: "quarter"},
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported layout: %s", layout)
	}
}

// GetSlotMetadata returns slot metadata for a layout (public method)
func (s *MashupService) GetSlotMetadata(layout string) ([]MashupSlotInfo, error) {
	return s.generateSlotMetadata(layout)
}

// GetAvailableLayouts returns all available mashup layouts
func (s *MashupService) GetAvailableLayouts() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":          "1Lx1R",
			"name":        "Split Vertical",
			"description": "Two equal vertical panels",
			"slots":       2,
			"icon":        "layout-split-vertical",
		},
		{
			"id":          "1Tx1B",
			"name":        "Split Horizontal",
			"description": "Two equal horizontal panels",
			"slots":       2,
			"icon":        "layout-split-horizontal",
		},
		{
			"id":          "1Lx2R",
			"name":        "One Left, Two Right",
			"description": "Large left panel with two stacked right panels",
			"slots":       3,
			"icon":        "layout-sidebar-right",
		},
		{
			"id":          "2Lx1R",
			"name":        "Two Left, One Right",
			"description": "Two stacked left panels with large right panel",
			"slots":       3,
			"icon":        "layout-sidebar-left",
		},
		{
			"id":          "2Tx1B",
			"name":        "Two Top, One Bottom",
			"description": "Two top panels with full-width bottom panel",
			"slots":       3,
			"icon":        "layout-header",
		},
		{
			"id":          "1Tx2B",
			"name":        "One Top, Two Bottom",
			"description": "Full-width top panel with two bottom panels",
			"slots":       3,
			"icon":        "layout-footer",
		},
		{
			"id":          "2x2",
			"name":        "Grid",
			"description": "Four equal quadrants",
			"slots":       4,
			"icon":        "layout-grid",
		},
	}
}