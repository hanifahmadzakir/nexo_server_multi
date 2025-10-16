package config

// ToolConfig defines the configuration for each Nexo tool.
type ToolConfig struct {
	ID   string // Unique ID for the API, e.g. "nexo-pistol"
	Name string // Descriptive name, e.g. "Nexo Pistol Grip"
	IP   string
	Port string
}

// GetTools return list of all tools to be connected by the server.
func GetTools() []ToolConfig {
	return []ToolConfig{
		{ID: "nexo-pistol", Name: "Nexo Pistol Grip", IP: "192.168.0.23", Port: "4545"},
		{ID: "nexo-angle", Name: "Nexo Angle Nutrunner", IP: "192.168.0.25", Port: "4545"},
		{ID: "nexo-angle-2", Name: "Nexo Angle 2nd Unit", IP: "192.168.0.50", Port: "4545"},
		// Tambahkan alat baru di sini.
	}
}