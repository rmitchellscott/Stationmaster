package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents a user account in the system
type User struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Username            string    `gorm:"uniqueIndex;not null" json:"username"`
	Email               string    `gorm:"uniqueIndex;not null" json:"email"`
	Password            string    `gorm:"not null" json:"-"` // Never return password in JSON
	IsAdmin             bool      `gorm:"default:false" json:"is_admin"`
	IsActive            bool      `gorm:"default:true" json:"is_active"`
	OnboardingCompleted bool      `gorm:"default:false" json:"onboarding_completed"`
	FirstName           string    `gorm:"size:100" json:"first_name,omitempty"`   // User's first name
	LastName            string    `gorm:"size:100" json:"last_name,omitempty"`    // User's last name
	Timezone            string    `gorm:"size:50;default:'UTC'" json:"timezone"` // User's preferred timezone (IANA format)
	Locale              string    `gorm:"size:10;default:'en-US'" json:"locale"` // User's preferred locale

	// Password reset
	ResetToken        string    `gorm:"index" json:"-"`
	ResetTokenExpires time.Time `json:"-"`

	// OIDC integration
	OidcSubject *string `gorm:"column:oidc_subject;uniqueIndex" json:"oidc_subject,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`

	// Associations
	APIKeys  []APIKey      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Sessions []UserSession `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// BeforeCreate sets UUID if not already set
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// APIKey represents an API key for a user
type APIKey struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string     `gorm:"not null" json:"name"`
	KeyHash   string     `gorm:"not null;index" json:"-"`            // Never return actual key
	KeyPrefix string     `gorm:"size:16;not null" json:"key_prefix"` // First 16 chars for display
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	// Association
	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (a *APIKey) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// UserSession represents a user's login session
type UserSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"not null;index" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_used"`
	UserAgent string    `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress string    `gorm:"size:45" json:"ip_address,omitempty"`

	// Association
	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (s *UserSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// SystemSetting represents system-wide configuration
type SystemSetting struct {
	Key         string     `gorm:"primaryKey" json:"key"`
	Value       string     `gorm:"type:text" json:"value"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	UpdatedAt   time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy   *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`

	// Association
	UpdatedByUser *User `gorm:"foreignKey:UpdatedBy" json:"-"`
}

// LoginAttempt represents a login attempt for rate limiting
type LoginAttempt struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	IPAddress   string    `gorm:"size:45;not null;index" json:"ip_address"`
	Username    string    `gorm:"index" json:"username,omitempty"`
	Success     bool      `gorm:"default:false" json:"success"`
	AttemptedAt time.Time `gorm:"default:CURRENT_TIMESTAMP;index" json:"attempted_at"`
	UserAgent   string    `gorm:"type:text" json:"user_agent,omitempty"`
}

func (l *LoginAttempt) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// BackupJob represents a background backup operation
type BackupJob struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AdminUserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"admin_user_id"`
	Status         string     `gorm:"size:50;not null;default:pending" json:"status"`
	Progress       int        `gorm:"default:0" json:"progress"`
	IncludeFiles   bool       `gorm:"default:true" json:"include_files"`
	IncludeConfigs bool       `gorm:"default:true" json:"include_configs"`
	UserIDs        string     `gorm:"type:text" json:"user_ids,omitempty"`
	FilePath       string     `gorm:"size:1000" json:"file_path,omitempty"`
	Filename       string     `gorm:"size:255" json:"filename,omitempty"`
	FileSize       int64      `json:"file_size,omitempty"`
	StatusMessage  string     `gorm:"type:text" json:"status_message,omitempty"`
	ErrorMessage   string     `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`

	// Association
	AdminUser User `gorm:"foreignKey:AdminUserID" json:"-"`
}

func (b *BackupJob) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// RestoreUpload represents an uploaded restore file waiting for confirmation
type RestoreUpload struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AdminUserID uuid.UUID `gorm:"type:uuid;not null;index" json:"admin_user_id"`
	Filename    string    `gorm:"size:255;not null" json:"filename"`
	FilePath    string    `gorm:"size:1000;not null" json:"file_path"`
	FileSize    int64     `json:"file_size"`
	Status      string    `gorm:"size:50;not null;default:uploaded" json:"status"`
	ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`

	// Association
	AdminUser User `gorm:"foreignKey:AdminUserID" json:"-"`
}

func (r *RestoreUpload) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// RestoreExtractionJob represents a background tar extraction operation for restore
type RestoreExtractionJob struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AdminUserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"admin_user_id"`
	RestoreUploadID uuid.UUID  `gorm:"type:uuid;not null;index" json:"restore_upload_id"`
	Status          string     `gorm:"size:50;not null;default:pending" json:"status"`
	Progress        int        `gorm:"default:0" json:"progress"`
	StatusMessage   string     `gorm:"type:text" json:"status_message,omitempty"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message,omitempty"`
	ExtractedPath   string     `gorm:"size:1000" json:"extracted_path,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`

	// Associations
	AdminUser     User          `gorm:"foreignKey:AdminUserID" json:"-"`
	RestoreUpload RestoreUpload `gorm:"foreignKey:RestoreUploadID;constraint:OnDelete:CASCADE" json:"-"`
}

func (r *RestoreExtractionJob) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// Device represents a TRMNL device that can be claimed by users
type Device struct {
	ID                      uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID                  *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`         // Nullable for unclaimed devices
	MacAddress              string     `gorm:"size:255;not null;uniqueIndex" json:"mac_address"` // Original MAC address from device
	FriendlyID              string     `gorm:"size:10;not null;uniqueIndex" json:"friendly_id"`  // Generated short ID like "917F0B"
	Name                    string     `gorm:"size:255" json:"name,omitempty"`                   // User-defined name, empty until claimed
	DeviceModelID           *uint      `gorm:"index" json:"device_model_id,omitempty"`           // Foreign key to DeviceModel.ID
	ManualModelOverride     bool       `gorm:"default:false" json:"manual_model_override"`       // True if model was manually set by user
	ReportedModelName       *string    `gorm:"size:100" json:"reported_model_name,omitempty"`    // Last model reported by device
	APIKey                  string     `gorm:"size:255;not null;index" json:"api_key"`
	IsClaimed               bool       `gorm:"default:false" json:"is_claimed"`
	FirmwareVersion         string     `gorm:"size:50" json:"firmware_version,omitempty"`
	BatteryVoltage          float64    `json:"battery_voltage,omitempty"`
	RSSI                    int        `json:"rssi,omitempty"`
	RefreshRate             int        `gorm:"default:1800" json:"refresh_rate"` // seconds
	AllowFirmwareUpdates    bool       `gorm:"default:true" json:"allow_firmware_updates"`
	LastSeen                *time.Time `json:"last_seen,omitempty"`
	LastPlaylistItemID      *uuid.UUID `gorm:"type:uuid;references:playlist_items(id)" json:"last_playlist_item_id,omitempty"` // Track last shown playlist item by UUID
	IsActive                bool       `gorm:"default:true" json:"is_active"`
	IsShareable             bool       `gorm:"default:false" json:"is_shareable"`                        // Whether this device can be mirrored by others
	MirrorSourceID          *uuid.UUID `gorm:"type:uuid;index" json:"mirror_source_id,omitempty"`        // ID of device being mirrored (nullable)
	MirrorSyncedAt          *time.Time `json:"mirror_synced_at,omitempty"`                               // Last time content was synced from source
	SleepEnabled            bool       `gorm:"default:false" json:"sleep_enabled"`                       // Whether sleep mode is active
	SleepStartTime          string     `gorm:"size:5" json:"sleep_start_time,omitempty"`                 // Start time in HH:MM format
	SleepEndTime            string     `gorm:"size:5" json:"sleep_end_time,omitempty"`                   // End time in HH:MM format
	SleepShowScreen         bool       `gorm:"default:true" json:"sleep_show_screen"`                    // Whether to show sleep image or last content
	FirmwareUpdateStartTime string     `gorm:"size:5;default:'00:00'" json:"firmware_update_start_time"` // Start time for firmware updates in HH:MM format
	FirmwareUpdateEndTime   string     `gorm:"size:5;default:'23:59'" json:"firmware_update_end_time"`   // End time for firmware updates in HH:MM format
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`

	// Associations
	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"-"`
	DeviceModel *DeviceModel `gorm:"foreignKey:DeviceModelID;constraint:OnDelete:SET NULL" json:"device_model,omitempty"`
	// MirrorSource association removed to avoid circular foreign key constraints during migration
	// Use MirrorSourceID field and fetch the source device manually when needed
	// Playlists relationship defined in Playlist model to avoid circular constraints
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// Plugin represents a system-wide plugin type (managed by admins)

// PrivatePluginWebhookData represents webhook data storage for private plugin instances
type PrivatePluginWebhookData struct {
	ID                 string                 `json:"id" gorm:"primaryKey"`
	PluginInstanceID   string                 `json:"plugin_instance_id" gorm:"index;not null"` // Changed from plugin_id to plugin_instance_id
	MergedData         datatypes.JSON         `json:"merged_data"`    // Final merged data ready for templates
	RawData            datatypes.JSON         `json:"raw_data"`       // Original webhook payload
	MergeStrategy      string                 `json:"merge_strategy" gorm:"size:20;default:'default'"`
	ReceivedAt         time.Time              `json:"received_at"`
	ContentType        string                 `json:"content_type"`
	ContentSize        int                    `json:"content_size"`
	SourceIP           string                 `json:"source_ip"`
}

// Playlist represents a collection of plugins for a specific device
type Playlist struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceID  uuid.UUID `gorm:"type:uuid;not null;index" json:"device_id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	IsDefault bool      `gorm:"default:false" json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	User          User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Device        Device         `gorm:"-:migration" json:"-"` // Skip constraint creation to avoid circular reference
	PlaylistItems []PlaylistItem `gorm:"foreignKey:PlaylistID;constraint:OnDelete:CASCADE" json:"-"`
}

func (pl *Playlist) BeforeCreate(tx *gorm.DB) error {
	if pl.ID == uuid.Nil {
		pl.ID = uuid.New()
	}
	return nil
}

// PlaylistItem represents a plugin within a playlist with ordering and visibility
type PlaylistItem struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PlaylistID       uuid.UUID `gorm:"type:uuid;not null;index" json:"playlist_id"`
	
	PluginInstanceID uuid.UUID `gorm:"type:uuid;not null;index" json:"plugin_instance_id"`
	
	OrderIndex       int       `gorm:"not null" json:"order_index"`
	IsVisible        bool      `gorm:"default:true" json:"is_visible"`
	Importance       bool      `gorm:"default:false" json:"importance"` // false=normal, true=important
	DurationOverride *int      `json:"duration_override,omitempty"`     // override default refresh rate
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Associations
	Playlist       Playlist       `gorm:"foreignKey:PlaylistID" json:"-"`
	PluginInstance PluginInstance `gorm:"foreignKey:PluginInstanceID" json:"plugin_instance"`
	Schedules      []Schedule     `gorm:"foreignKey:PlaylistItemID;constraint:OnDelete:CASCADE" json:"schedules"`
}

func (pi *PlaylistItem) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == uuid.Nil {
		pi.ID = uuid.New()
	}
	return nil
}

// Schedule represents a time-based schedule for when a playlist item should be active
type Schedule struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PlaylistItemID uuid.UUID `gorm:"type:uuid;not null;index" json:"playlist_item_id"`
	Name           string    `gorm:"size:255" json:"name,omitempty"`    // User-friendly name for this schedule
	DayMask        int       `gorm:"not null" json:"day_mask"`          // Bitmask: Sunday=1, Monday=2, Tuesday=4, etc.
	StartTime      string    `gorm:"size:8;not null" json:"start_time"` // HH:MM:SS format
	EndTime        string    `gorm:"size:8;not null" json:"end_time"`   // HH:MM:SS format
	Timezone       string    `gorm:"size:50;default:'UTC'" json:"timezone"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Associations
	PlaylistItem PlaylistItem `gorm:"foreignKey:PlaylistItemID" json:"-"`
}

func (s *Schedule) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// PluginDefinition represents a unified plugin definition (system, private, or public)
type PluginDefinition struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PluginType string     `gorm:"size:20;not null" json:"plugin_type"` // "system", "private", "public"
	OwnerID    *uuid.UUID `gorm:"type:uuid;index" json:"owner_id"`     // NULL for system plugins, user_id for private/public
	
	// Core fields
	Identifier         string `gorm:"size:255;not null" json:"identifier"`          // Type string for system, UUID for private
	Name              string `gorm:"size:255;not null" json:"name"`
	Description       string `gorm:"type:text" json:"description"`
	Version           string `gorm:"size:50" json:"version"`
	Author            string `gorm:"size:255" json:"author"`
	ConfigSchema      string `gorm:"type:text" json:"config_schema"` // JSON schema for settings
	RequiresProcessing bool   `gorm:"default:false" json:"requires_processing"`
	
	// Private/Public plugin specific fields (NULL for system plugins)
	MarkupFull      *string        `gorm:"type:text" json:"markup_full,omitempty"`
	MarkupHalfVert  *string        `gorm:"type:text" json:"markup_half_vert,omitempty"`
	MarkupHalfHoriz *string        `gorm:"type:text" json:"markup_half_horiz,omitempty"`
	MarkupQuadrant  *string        `gorm:"type:text" json:"markup_quadrant,omitempty"`
	SharedMarkup    *string        `gorm:"type:text" json:"shared_markup,omitempty"`
	DataStrategy    *string        `gorm:"size:50" json:"data_strategy,omitempty"`      // webhook, polling, static
	PollingConfig   datatypes.JSON `json:"polling_config,omitempty"`   // URLs, headers, body, intervals, etc.
	FormFields      datatypes.JSON `json:"form_fields"`                // YAML form field definitions converted to JSON schema
	
	// Publishing (for future public plugins)
	IsPublished bool       `gorm:"default:false" json:"is_published"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	
	// Screen options (for private plugins)
	RemoveBleedMargin *bool          `gorm:"default:false" json:"remove_bleed_margin,omitempty"` // Nullable for backward compatibility
	EnableDarkMode    *bool          `gorm:"default:false" json:"enable_dark_mode,omitempty"`    // Nullable for backward compatibility
	SampleData        datatypes.JSON `json:"sample_data,omitempty"`                              // JSON sample data for preview/testing
	
	// Schema versioning for form field changes
	SchemaVersion int `gorm:"default:1" json:"schema_version"` // Increments when FormFields change
	
	// Meta
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	
	// Associations
	Owner     *User            `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Instances []PluginInstance `gorm:"foreignKey:PluginDefinitionID;constraint:OnDelete:CASCADE" json:"-"`
}

func (pd *PluginDefinition) BeforeCreate(tx *gorm.DB) error {
	if pd.ID == uuid.Nil {
		pd.ID = uuid.New()
	}
	return nil
}

// PluginInstance represents a user's instance of any plugin type with specific settings
type PluginInstance struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID             uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	PluginDefinitionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"plugin_definition_id"`
	
	Name            string         `gorm:"size:255;not null" json:"name"`        // User-defined name for this instance
	Settings        datatypes.JSON `gorm:"type:text" json:"settings"`           // JSON settings specific to this instance
	RefreshInterval int           `gorm:"default:3600" json:"refresh_interval"` // Refresh interval in seconds
	IsActive        bool          `gorm:"default:true" json:"is_active"`
	
	// Schema version tracking for config update detection
	LastSchemaVersion   int  `gorm:"default:1" json:"last_schema_version"`      // Schema version this instance was last updated against
	NeedsConfigUpdate   bool `gorm:"default:false" json:"needs_config_update"`  // Flag when parent plugin schema changes
	
	CreatedAt       time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	
	// Associations
	User             User              `gorm:"foreignKey:UserID" json:"-"`
	PluginDefinition PluginDefinition  `gorm:"foreignKey:PluginDefinitionID" json:"plugin_definition"`
	PlaylistItems    []PlaylistItem    `gorm:"foreignKey:PluginInstanceID;constraint:OnDelete:CASCADE" json:"-"`
	RenderedContent  []RenderedContent `gorm:"foreignKey:PluginInstanceID;constraint:OnDelete:CASCADE" json:"-"`
}

func (pi *PluginInstance) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == uuid.Nil {
		pi.ID = uuid.New()
	}
	return nil
}

// DeviceLog represents a log entry from a device
type DeviceLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID  uuid.UUID `gorm:"type:uuid;not null;index" json:"device_id"`
	LogData   string    `gorm:"type:text;not null" json:"log_data"`  // JSON log content
	Level     string    `gorm:"size:20;default:'info'" json:"level"` // info, warn, error, debug
	Timestamp time.Time `gorm:"not null;index" json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`

	// Associations
	Device Device `gorm:"foreignKey:DeviceID" json:"-"`
}

func (dl *DeviceLog) BeforeCreate(tx *gorm.DB) error {
	if dl.ID == uuid.Nil {
		dl.ID = uuid.New()
	}
	if dl.Timestamp.IsZero() {
		dl.Timestamp = time.Now()
	}
	return nil
}

// FirmwareVersion represents a firmware version available for devices
type FirmwareVersion struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Version          string    `gorm:"size:50;not null;uniqueIndex" json:"version"`
	ReleaseNotes     string    `gorm:"type:text" json:"release_notes,omitempty"`
	DownloadURL      string    `gorm:"size:1000;not null" json:"download_url"`
	FileSize         int64     `json:"file_size,omitempty"`
	FilePath         string    `gorm:"size:1000" json:"file_path,omitempty"` // Local storage path
	SHA256           string    `gorm:"size:64" json:"sha256,omitempty"`
	IsLatest         bool      `gorm:"default:false" json:"is_latest"`
	IsDownloaded     bool      `gorm:"default:false" json:"is_downloaded"`
	DownloadStatus   string    `gorm:"size:20;default:'pending'" json:"download_status"` // pending, downloading, downloaded, failed
	DownloadProgress int       `gorm:"default:0" json:"download_progress"`               // 0-100
	DownloadError    string    `gorm:"type:text" json:"download_error,omitempty"`        // Error message if download failed
	ReleasedAt       time.Time `json:"released_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (f *FirmwareVersion) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// DeviceModel represents a device model with its capabilities
type DeviceModel struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelName      string     `gorm:"size:100;not null;index" json:"model_name"` // Not unique - multiple versions allowed
	DisplayName    string     `gorm:"size:200;not null" json:"display_name"`
	Description    string     `gorm:"type:text" json:"description,omitempty"`
	ScreenWidth    int        `gorm:"not null" json:"screen_width"`
	ScreenHeight   int        `gorm:"not null" json:"screen_height"`
	ColorDepth     int        `gorm:"default:1" json:"color_depth"` // 1=monochrome, 8=grayscale, 24=color
	BitDepth       int        `gorm:"default:1" json:"bit_depth"`   // Actual bit depth of the display
	HasWiFi        bool       `gorm:"default:true" json:"has_wifi"`
	HasBattery     bool       `gorm:"default:true" json:"has_battery"`
	HasButtons     int        `gorm:"default:0" json:"has_buttons"`            // Number of buttons
	Capabilities   string     `gorm:"type:text" json:"capabilities,omitempty"` // JSON array of capabilities
	MinFirmware    string     `gorm:"size:50" json:"min_firmware,omitempty"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	ApiLastSeenAt  *time.Time `json:"api_last_seen_at,omitempty"` // Track when last seen in API
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at,omitempty"` // Soft delete
}

// Note: No BeforeCreate needed for auto-increment ID


// FirmwareUpdateJob - REMOVED: Using automatic updates now instead of job-based system

// RenderedContent represents a cached rendered image for a plugin
type RenderedContent struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PluginInstanceID uuid.UUID  `gorm:"type:uuid;not null;index" json:"plugin_instance_id"`
	DeviceID         *uuid.UUID `gorm:"type:uuid;index" json:"device_id,omitempty"` // Nullable for backward compatibility
	
	Width        int       `gorm:"not null" json:"width"`
	Height       int       `gorm:"not null" json:"height"`
	BitDepth     int       `gorm:"not null" json:"bit_depth"`
	ImagePath    string    `gorm:"size:1000;not null" json:"image_path"`
	FileSize     int64     `json:"file_size"`
	ContentHash  *string   `gorm:"size:64" json:"content_hash,omitempty"`
	RenderedAt   time.Time `gorm:"not null;index" json:"rendered_at"`
	CreatedAt    time.Time `json:"created_at"`
	
	// Enhanced content lifecycle tracking
	LastCheckedAt  *time.Time `gorm:"index" json:"last_checked_at,omitempty"` // Track hash comparisons even when not saving
	PreviousHash   *string    `gorm:"size:64" json:"previous_hash,omitempty"`  // Store previous content hash for debugging
	RenderAttempts int        `gorm:"default:0" json:"render_attempts"`        // Track render failures
	
	// Associations  
	PluginInstance PluginInstance `gorm:"foreignKey:PluginInstanceID" json:"-"`
	Device         *Device        `gorm:"foreignKey:DeviceID" json:"-"`
}

func (rc *RenderedContent) BeforeCreate(tx *gorm.DB) error {
	if rc.ID == uuid.Nil {
		rc.ID = uuid.New()
	}
	return nil
}

// RenderQueue represents pending render jobs for plugins
type RenderQueue struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	
	PluginInstanceID uuid.UUID `gorm:"type:uuid;not null;index" json:"plugin_instance_id"`
	
	Priority         int        `gorm:"default:0;index" json:"priority"` // Higher number = higher priority
	ScheduledFor     time.Time  `gorm:"not null;index" json:"scheduled_for"`
	Status           string     `gorm:"size:50;default:pending;index" json:"status"` // pending, processing, completed, failed
	IndependentRender bool       `gorm:"default:false" json:"independent_render"` // true = don't reschedule after completion
	Attempts         int        `gorm:"default:0" json:"attempts"`
	LastAttempt      *time.Time `json:"last_attempt,omitempty"`
	ErrorMessage     string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Associations
	PluginInstance PluginInstance `gorm:"foreignKey:PluginInstanceID" json:"-"`
}

func (rq *RenderQueue) BeforeCreate(tx *gorm.DB) error {
	if rq.ID == uuid.Nil {
		rq.ID = uuid.New()
	}
	return nil
}

// GetAllModels returns all models for auto-migration
func GetAllModels() []interface{} {
	return []interface{}{
		&User{},
		&APIKey{},
		&UserSession{},
		&SystemSetting{},
		&LoginAttempt{},
		&BackupJob{},
		&RestoreUpload{},
		&RestoreExtractionJob{},
		&DeviceModel{}, // Must come before Device due to foreign key reference
		&Device{},
		
		&PrivatePluginWebhookData{}, // Webhook data for plugin instances
		
		// New unified plugin models
		&PluginDefinition{}, // Must come after User due to foreign key reference
		&PluginInstance{},   // Must come after PluginDefinition and User
		
		&Playlist{},
		&PlaylistItem{},
		&Schedule{},
		&DeviceLog{},
		&FirmwareVersion{},
		&RenderedContent{},
		&RenderQueue{},
		// &FirmwareUpdateJob{}, // Removed - using automatic updates
	}
}
