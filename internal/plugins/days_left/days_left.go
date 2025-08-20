package days_left

import (
	"fmt"
	"time"

	"github.com/rmitchellscott/stationmaster/internal/plugins"
)

// DaysLeftPlugin implements a data plugin that calculates days remaining until a target date
type DaysLeftPlugin struct{}

// Type returns the plugin type identifier
func (p *DaysLeftPlugin) Type() string {
	return "days_left_until"
}

// PluginType returns that this is a data plugin
func (p *DaysLeftPlugin) PluginType() plugins.PluginType {
	return plugins.PluginTypeData
}

// Name returns the human-readable name
func (p *DaysLeftPlugin) Name() string {
	return "Days Left Until"
}

// Description returns the plugin description
func (p *DaysLeftPlugin) Description() string {
	return "Displays the number of days passed and remaining between two dates with a progress bar"
}

// RequiresProcessing returns true since this plugin needs HTML rendering
func (p *DaysLeftPlugin) RequiresProcessing() bool {
	return true
}

// ConfigSchema returns the JSON schema for configuration
func (p *DaysLeftPlugin) ConfigSchema() string {
	return `{
		"type": "object",
		"properties": {
			"start_date": {
				"type": "string",
				"title": "Start Date",
				"description": "The start date (YYYY-MM-DD)",
				"format": "date"
			},
			"end_date": {
				"type": "string",
				"title": "End Date",
				"description": "The end date (YYYY-MM-DD)",
				"format": "date"
			},
			"message": {
				"type": "string",
				"title": "Custom Message",
				"description": "Optional custom message to display",
				"default": ""
			},
			"show_days_passed": {
				"type": "boolean",
				"title": "Show Days Passed",
				"description": "Whether to show the number of days that have passed",
				"default": true
			},
			"show_days_left": {
				"type": "boolean",
				"title": "Show Days Left",
				"description": "Whether to show the number of days remaining",
				"default": true
			},
			"show_percentage": {
				"type": "boolean",
				"title": "Show Percentage",
				"description": "Whether to show the percentage completed",
				"default": true
			}
		},
		"required": ["start_date", "end_date"]
	}`
}

// DataSchema returns the schema of the data structure returned
func (p *DaysLeftPlugin) DataSchema() string {
	return `{
		"type": "object",
		"properties": {
			"days_passed": {"type": "integer"},
			"days_left": {"type": "integer"},
			"total_days": {"type": "integer"},
			"percent_passed": {"type": "number"},
			"message": {"type": "string"},
			"start_date": {"type": "string"},
			"end_date": {"type": "string"},
			"show_days_passed": {"type": "boolean"},
			"show_days_left": {"type": "boolean"},
			"show_percentage": {"type": "boolean"},
			"is_complete": {"type": "boolean"},
			"is_future": {"type": "boolean"}
		}
	}`
}

// RenderTemplate returns the HTML template for rendering the data using TRMNL framework
func (p *DaysLeftPlugin) RenderTemplate() string {
	return `
<div class="layout">
	<div class="columns">
		<div class="column gap--xxlarge">
			{{if .message}}
				<div class="title text--center">{{.message}}</div>
			{{end}}
			
			{{if .is_future}}
				<div class="text--center">
					<div class="value value--large">Event hasn't started yet</div>
					<div class="label">Starts: {{.start_date}}</div>
				</div>
			{{else if .is_complete}}
				<div class="text--center">
					<div class="value value--large">Event completed!</div>
					<div class="label">Ended: {{.end_date}}</div>
				</div>
			{{else}}
				{{if or .show_days_passed .show_days_left}}
					<div class="metrics metrics--column-3 days-stats">
						{{if .show_days_passed}}
							<div class="metric">
								<span class="value value--xxxlarge">{{.days_passed}}</span>
								<span class="label label--underline">Days Passed</span>
							</div>
						{{end}}
						
						{{if .show_days_left}}
							<div class="metric">
								<span class="value value--xxxlarge">{{.days_left}}</span>
								<span class="label label--underline">Days Left</span>
							</div>
						{{end}}
					</div>
				{{end}}
				
				{{if .show_percentage}}
					<div class="text--center">
						<div class="progress-container" style="width: 300px; margin: 20px auto;">
							<div class="progress-track" style="background: #e9ecef; height: 20px; border-radius: 10px; overflow: hidden;">
								<div class="progress-bar" style="width: {{.percent_passed}}%; background: linear-gradient(45deg, #007bff, #0056b3); height: 100%; transition: width 0.3s ease;"></div>
							</div>
							<div class="progress-label" style="text-align: center; margin-top: 5px; font-size: 14px;">
								{{printf "%.0f" .percent_passed}}% Complete
							</div>
						</div>
					</div>
				{{end}}
				
				<div class="text--center">
					<div class="label">{{.start_date}} to {{.end_date}}</div>
					<div class="label">{{.total_days}} total days</div>
				</div>
			{{end}}
		</div>
	</div>
</div>

<div class="title_bar">
	<span class="title">Days Left Until</span>
	{{if .message}}
		<span class="instance">{{.message}}</span>
	{{end}}
</div>
	`
}

// Validate validates the plugin settings
func (p *DaysLeftPlugin) Validate(settings map[string]interface{}) error {
	startDate, ok := settings["start_date"].(string)
	if !ok || startDate == "" {
		return fmt.Errorf("start_date is required")
	}

	endDate, ok := settings["end_date"].(string)
	if !ok || endDate == "" {
		return fmt.Errorf("end_date is required")
	}

	// Parse dates to validate format
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("start_date must be in YYYY-MM-DD format")
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("end_date must be in YYYY-MM-DD format")
	}

	if !end.After(start) {
		return fmt.Errorf("end_date must be after start_date")
	}

	return nil
}

// Process executes the plugin logic
func (p *DaysLeftPlugin) Process(ctx plugins.PluginContext) (plugins.PluginResponse, error) {
	// Parse dates
	startDate, err := time.Parse("2006-01-02", ctx.GetStringSetting("start_date", ""))
	if err != nil {
		return plugins.CreateErrorResponse("Invalid start_date format"),
			fmt.Errorf("failed to parse start_date: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", ctx.GetStringSetting("end_date", ""))
	if err != nil {
		return plugins.CreateErrorResponse("Invalid end_date format"),
			fmt.Errorf("failed to parse end_date: %w", err)
	}

	// Calculate with current time in user's timezone
	now := time.Now()
	if ctx.Device != nil && ctx.Device.UserID != nil {
		// TODO: Get user timezone from database
		// For now, use UTC
	}

	// Calculate days
	totalDuration := endDate.Sub(startDate)
	totalDays := int(totalDuration.Hours() / 24)

	passedDuration := now.Sub(startDate)
	daysPassed := int(passedDuration.Hours() / 24)

	remainingDuration := endDate.Sub(now)
	daysLeft := int(remainingDuration.Hours() / 24)

	// Calculate percentage
	var percentPassed float64
	if totalDays > 0 {
		percentPassed = (float64(daysPassed) / float64(totalDays)) * 100
	}

	// Ensure percentage is within bounds
	if percentPassed < 0 {
		percentPassed = 0
	} else if percentPassed > 100 {
		percentPassed = 100
	}

	// Determine state
	isFuture := now.Before(startDate)
	isComplete := now.After(endDate)

	// Prepare data
	data := map[string]interface{}{
		"days_passed":      max(0, daysPassed),
		"days_left":        max(0, daysLeft),
		"total_days":       totalDays,
		"percent_passed":   percentPassed,
		"message":          ctx.GetStringSetting("message", ""),
		"start_date":       startDate.Format("2006-01-02"),
		"end_date":         endDate.Format("2006-01-02"),
		"show_days_passed": ctx.GetBoolSetting("show_days_passed", true),
		"show_days_left":   ctx.GetBoolSetting("show_days_left", true),
		"show_percentage":  ctx.GetBoolSetting("show_percentage", true),
		"is_complete":      isComplete,
		"is_future":        isFuture,
	}

	// Refresh rate: daily updates
	refreshRate := 86400 // 24 hours

	return plugins.CreateDataResponse(data, p.RenderTemplate(), refreshRate), nil
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Register the plugin when this package is imported
func init() {
	plugins.Register(&DaysLeftPlugin{})
}