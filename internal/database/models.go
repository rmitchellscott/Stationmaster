package database

import (
	"time"

	"github.com/google/uuid"
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
	Timezone            string    `gorm:"size:50;default:'UTC'" json:"timezone"` // User's preferred timezone (IANA format)

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
	ID                   uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID               *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`         // Nullable for unclaimed devices
	MacAddress           string     `gorm:"size:255;not null;uniqueIndex" json:"mac_address"` // Original MAC address from device
	FriendlyID           string     `gorm:"size:10;not null;uniqueIndex" json:"friendly_id"`  // Generated short ID like "917F0B"
	Name                 string     `gorm:"size:255" json:"name,omitempty"`                   // User-defined name, empty until claimed
	ModelName            *string    `gorm:"size:100;index" json:"model_name,omitempty"`       // Device model identifier
	ManualModelOverride  bool       `gorm:"default:false" json:"manual_model_override"`       // True if model was manually set by user
	ReportedModelName    *string    `gorm:"size:100" json:"reported_model_name,omitempty"`    // Last model reported by device
	APIKey               string     `gorm:"size:255;not null;index" json:"api_key"`
	IsClaimed            bool       `gorm:"default:false" json:"is_claimed"`
	FirmwareVersion      string     `gorm:"size:50" json:"firmware_version,omitempty"`
	BatteryVoltage       float64    `json:"battery_voltage,omitempty"`
	RSSI                 int        `json:"rssi,omitempty"`
	RefreshRate          int        `gorm:"default:1800" json:"refresh_rate"` // seconds
	AllowFirmwareUpdates bool       `gorm:"default:true" json:"allow_firmware_updates"`
	LastSeen             *time.Time `json:"last_seen,omitempty"`
	LastPlaylistIndex    int        `gorm:"default:0" json:"last_playlist_index"` // Track last shown playlist item
	IsActive             bool       `gorm:"default:true" json:"is_active"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Associations
	User        *User        `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"-"`
	DeviceModel *DeviceModel `gorm:"references:ModelName;foreignKey:ModelName" json:"device_model,omitempty"`
	// Playlists relationship defined in Playlist model to avoid circular constraints
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// Plugin represents a system-wide plugin type (managed by admins)
type Plugin struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string    `gorm:"size:255;not null;uniqueIndex" json:"name"`
	Type         string    `gorm:"size:100;not null" json:"type"`
	Description  string    `gorm:"type:text" json:"description"`
	ConfigSchema string    `gorm:"type:text" json:"config_schema"` // JSON schema for plugin settings
	Version      string    `gorm:"size:50" json:"version"`
	Author       string    `gorm:"size:255" json:"author,omitempty"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Associations
	UserPlugins []UserPlugin `gorm:"foreignKey:PluginID;constraint:OnDelete:CASCADE" json:"-"`
}

func (p *Plugin) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// UserPlugin represents a user's instance of a plugin with specific settings
type UserPlugin struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	PluginID  uuid.UUID `gorm:"type:uuid;not null;index" json:"plugin_id"`
	Name      string    `gorm:"size:255;not null" json:"name"` // User-defined name for this instance
	Settings  string    `gorm:"type:text" json:"settings"`     // JSON settings specific to this instance
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	User          User           `gorm:"foreignKey:UserID" json:"-"`
	Plugin        Plugin         `gorm:"foreignKey:PluginID" json:"plugin"`
	PlaylistItems []PlaylistItem `gorm:"foreignKey:UserPluginID;constraint:OnDelete:CASCADE" json:"-"`
}

func (up *UserPlugin) BeforeCreate(tx *gorm.DB) error {
	if up.ID == uuid.Nil {
		up.ID = uuid.New()
	}
	return nil
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
	UserPluginID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_plugin_id"`
	OrderIndex       int       `gorm:"not null" json:"order_index"`
	IsVisible        bool      `gorm:"default:true" json:"is_visible"`
	Importance       bool      `gorm:"default:false" json:"importance"` // false=normal, true=important
	DurationOverride *int      `json:"duration_override,omitempty"`     // override default refresh rate
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Associations
	Playlist   Playlist   `gorm:"foreignKey:PlaylistID" json:"-"`
	UserPlugin UserPlugin `gorm:"foreignKey:UserPluginID" json:"user_plugin"`
	Schedules  []Schedule `gorm:"foreignKey:PlaylistItemID;constraint:OnDelete:CASCADE" json:"schedules"`
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
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ModelName    string    `gorm:"size:100;not null;uniqueIndex" json:"model_name"`
	DisplayName  string    `gorm:"size:200;not null" json:"display_name"`
	Description  string    `gorm:"type:text" json:"description,omitempty"`
	ScreenWidth  int       `gorm:"not null" json:"screen_width"`
	ScreenHeight int       `gorm:"not null" json:"screen_height"`
	ColorDepth   int       `gorm:"default:1" json:"color_depth"` // 1=monochrome, 8=grayscale, 24=color
	BitDepth     int       `gorm:"default:1" json:"bit_depth"`   // Actual bit depth of the display
	HasWiFi      bool      `gorm:"default:true" json:"has_wifi"`
	HasBattery   bool      `gorm:"default:true" json:"has_battery"`
	HasButtons   int       `gorm:"default:0" json:"has_buttons"`            // Number of buttons
	Capabilities string    `gorm:"type:text" json:"capabilities,omitempty"` // JSON array of capabilities
	MinFirmware  string    `gorm:"size:50" json:"min_firmware,omitempty"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (d *DeviceModel) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// FirmwareUpdateJob - REMOVED: Using automatic updates now instead of job-based system

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
		&Device{},
		&Plugin{},
		&UserPlugin{},
		&Playlist{},
		&PlaylistItem{},
		&Schedule{},
		&DeviceLog{},
		&FirmwareVersion{},
		// &DeviceModel{}, // Managed manually in migrations to avoid foreign key constraint issues
		// &FirmwareUpdateJob{}, // Removed - using automatic updates
	}
}
