package mashup

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

func createTestPluginContext() plugins.PluginContext {
	return plugins.PluginContext{
		Device: &database.Device{
			DeviceModel: &database.DeviceModel{
				ScreenWidth:  800,
				ScreenHeight: 480,
			},
		},
		PluginInstance: &database.PluginInstance{
			ID: uuid.New(),
		},
		User: &database.User{},
		Settings: make(map[string]interface{}),
	}
}

func TestBuildMashupHTML_MultiPass(t *testing.T) {
	plugin := &MashupPlugin{
		definition: &database.PluginDefinition{
			Name: "Test Mashup",
		},
		instance: &database.PluginInstance{},
	}

	renderedSlots := map[string]string{
		"left":  "<div class=\"test-left\">Left Content</div>",
		"right": "<div class=\"test-right\">Right Content</div>",
	}

	slotConfig := []database.MashupSlotInfo{
		{
			Position:     "left",
			ViewClass:    "view--half_vertical",
			DisplayName:  "Left Panel",
			RequiredSize: "half",
		},
		{
			Position:     "right",
			ViewClass:    "view--half_vertical",
			DisplayName:  "Right Panel",
			RequiredSize: "half",
		},
	}

	ctx := createTestPluginContext()

	// Test rendering with left slot active
	htmlLeft := plugin.buildMashupHTML("1Lx1R", renderedSlots, slotConfig, ctx, "left")

	if !strings.Contains(htmlLeft, "test-left") {
		t.Error("buildMashupHTML() with activeSlot=left should contain left content")
	}

	if strings.Contains(htmlLeft, "test-right") {
		t.Error("buildMashupHTML() with activeSlot=left should NOT contain right content")
	}

	if !strings.Contains(htmlLeft, `id="slot-right"`) {
		t.Error("buildMashupHTML() should contain empty right slot div")
	}

	// Test rendering with right slot active
	htmlRight := plugin.buildMashupHTML("1Lx1R", renderedSlots, slotConfig, ctx, "right")

	if !strings.Contains(htmlRight, "test-right") {
		t.Error("buildMashupHTML() with activeSlot=right should contain right content")
	}

	if strings.Contains(htmlRight, "test-left") {
		t.Error("buildMashupHTML() with activeSlot=right should NOT contain left content")
	}

	if !strings.Contains(htmlRight, `id="slot-left"`) {
		t.Error("buildMashupHTML() should contain empty left slot div")
	}
}

