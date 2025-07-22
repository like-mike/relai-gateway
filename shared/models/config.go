package models

// Config represents the main application configuration
type Config struct {
	App          AppConfig              `yaml:"app"`
	Themes       map[string]Theme       `yaml:"themes"`
	ActiveTheme  string                 `yaml:"active_theme"`
	UI           UIConfig               `yaml:"ui"`
	Features     FeatureFlags           `yaml:"features"`
}

// AppConfig contains basic application information
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
	Description string `yaml:"description"`
}

// Theme represents a UI theme configuration
type Theme struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Primary     map[string]string      `yaml:"primary"`
	Secondary   map[string]string      `yaml:"secondary"`
	Accent      AccentColors           `yaml:"accent"`
	Branding    BrandingConfig         `yaml:"branding"`
}

// AccentColors defines semantic colors for the theme
type AccentColors struct {
	Success string `yaml:"success"`
	Warning string `yaml:"warning"`
	Error   string `yaml:"error"`
	Info    string `yaml:"info"`
}

// BrandingConfig contains branding assets
type BrandingConfig struct {
	Logo     string `yaml:"logo"`
	Favicon  string `yaml:"favicon"`
	Wordmark string `yaml:"wordmark"`
	Company  string `yaml:"company"`
}

// UIConfig contains UI-specific settings
type UIConfig struct {
	Sidebar    SidebarConfig    `yaml:"sidebar"`
	Header     HeaderConfig     `yaml:"header"`
	Footer     FooterConfig     `yaml:"footer"`
	Animations AnimationConfig  `yaml:"animations"`
}

// SidebarConfig defines sidebar behavior
type SidebarConfig struct {
	Width          string `yaml:"width"`
	CollapsedWidth string `yaml:"collapsed_width"`
}

// HeaderConfig defines header behavior
type HeaderConfig struct {
	Height       string `yaml:"height"`
	ShowLogo     bool   `yaml:"show_logo"`
	ShowUserMenu bool   `yaml:"show_user_menu"`
}

// FooterConfig defines footer behavior
type FooterConfig struct {
	Show bool   `yaml:"show"`
	Text string `yaml:"text"`
}

// AnimationConfig defines animation settings
type AnimationConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Duration string `yaml:"duration"`
}

// FeatureFlags defines toggleable features
type FeatureFlags struct {
	ThemeSwitching   bool `yaml:"theme_switching"`
	DarkMode         bool `yaml:"dark_mode"`
	AdvancedSettings bool `yaml:"advanced_settings"`
	Analytics        bool `yaml:"analytics"`
}

// ThemeContextData represents theme data for templates
type ThemeContextData struct {
	Config      *Config `json:"config"`
	ActiveTheme *Theme  `json:"active_theme"`
	ThemeKey    string  `json:"theme_key"`
}
