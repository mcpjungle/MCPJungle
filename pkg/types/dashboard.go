package types

type DashboardStatus string

const (
	DashboardStatusRunning  DashboardStatus = "running"
	DashboardStatusDegraded DashboardStatus = "degraded"
	DashboardStatusUnknown  DashboardStatus = "unknown"
)

type DashboardServerStatus string

const (
	DashboardServerStatusConnected DashboardServerStatus = "connected"
	DashboardServerStatusReachable DashboardServerStatus = "reachable"
	DashboardServerStatusFailed    DashboardServerStatus = "failed"
	DashboardServerStatusUnknown   DashboardServerStatus = "unknown"
)

type DashboardEndpoint struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type DashboardEmptyState struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Commands    []string `json:"commands,omitempty"`
}

type DashboardOverviewResponse struct {
	Status          DashboardStatus      `json:"status"`
	Mode            string               `json:"mode"`
	Version         string               `json:"version"`
	Endpoints       []DashboardEndpoint  `json:"endpoints"`
	ServerCount     int                  `json:"server_count"`
	ToolCount       int                  `json:"tool_count"`
	PromptCount     int                  `json:"prompt_count"`
	ResourceCount   int                  `json:"resource_count"`
	EmptyState      *DashboardEmptyState `json:"empty_state,omitempty"`
	Troubleshooting []string             `json:"troubleshooting,omitempty"`
}

type DashboardServerConfigSummary struct {
	Kind             string   `json:"kind"`
	Target           string   `json:"target,omitempty"`
	Command          string   `json:"command,omitempty"`
	ArgumentCount    int      `json:"argument_count,omitempty"`
	EnvKeys          []string `json:"env_keys,omitempty"`
	HeaderKeys       []string `json:"header_keys,omitempty"`
	SessionMode      string   `json:"session_mode,omitempty"`
	Description      string   `json:"description,omitempty"`
	SanitizedSummary string   `json:"sanitized_summary"`
}

type DashboardServer struct {
	Name               string                       `json:"name"`
	Transport          string                       `json:"transport"`
	Enabled            bool                         `json:"enabled"`
	Status             DashboardServerStatus        `json:"status"`
	ToolCount          int                          `json:"tool_count"`
	PromptCount        int                          `json:"prompt_count"`
	ResourceCount      int                          `json:"resource_count"`
	LastDiscoveredAt   string                       `json:"last_discovered_at,omitempty"`
	UpdatedAt          string                       `json:"updated_at,omitempty"`
	ConnectionSummary  string                       `json:"connection_summary"`
	ConfigSummary      DashboardServerConfigSummary `json:"config_summary"`
	NamespacedExamples []string                     `json:"namespaced_examples,omitempty"`
}

type DashboardServersResponse struct {
	Servers    []DashboardServer    `json:"servers"`
	EmptyState *DashboardEmptyState `json:"empty_state,omitempty"`
}

type DashboardTool struct {
	Name           string         `json:"name"`
	CanonicalName  string         `json:"canonical_name"`
	Server         string         `json:"server"`
	Description    string         `json:"description"`
	Enabled        bool           `json:"enabled"`
	ServerEnabled  bool           `json:"server_enabled"`
	InputSchema    map[string]any `json:"input_schema,omitempty"`
	InputPreview   string         `json:"input_preview,omitempty"`
	Transport      string         `json:"transport,omitempty"`
	ServerStatus   string         `json:"server_status,omitempty"`
	AnnotationKeys []string       `json:"annotation_keys,omitempty"`
}

type DashboardToolsResponse struct {
	Tools      []DashboardTool      `json:"tools"`
	EmptyState *DashboardEmptyState `json:"empty_state,omitempty"`
}

type DashboardPrompt struct {
	Name             string           `json:"name"`
	CanonicalName    string           `json:"canonical_name"`
	Server           string           `json:"server"`
	Description      string           `json:"description"`
	Enabled          bool             `json:"enabled"`
	ServerEnabled    bool             `json:"server_enabled"`
	Arguments        []map[string]any `json:"arguments,omitempty"`
	ArgumentsPreview string           `json:"arguments_preview,omitempty"`
	Transport        string           `json:"transport,omitempty"`
	ServerStatus     string           `json:"server_status,omitempty"`
}

type DashboardPromptsResponse struct {
	Prompts    []DashboardPrompt    `json:"prompts"`
	EmptyState *DashboardEmptyState `json:"empty_state,omitempty"`
}

type DashboardResource struct {
	URI          string `json:"uri"`
	Name         string `json:"name"`
	Server       string `json:"server"`
	Description  string `json:"description"`
	MIMEType     string `json:"mime_type,omitempty"`
	Enabled      bool   `json:"enabled"`
	Transport    string `json:"transport,omitempty"`
	ServerStatus string `json:"server_status,omitempty"`
}

type DashboardResourcesResponse struct {
	Resources  []DashboardResource  `json:"resources"`
	EmptyState *DashboardEmptyState `json:"empty_state,omitempty"`
}

type DashboardDiagnosticsResponse struct {
	Version              string               `json:"version"`
	Mode                 string               `json:"mode"`
	ConfigSource         string               `json:"config_source,omitempty"`
	ConfigPath           string               `json:"config_path,omitempty"`
	Database             string               `json:"database"`
	EnabledTransports    []string             `json:"enabled_transports"`
	MetricsEndpoint      string               `json:"metrics_endpoint,omitempty"`
	PrimaryEndpoint      string               `json:"primary_endpoint"`
	TroubleshootingHints []string             `json:"troubleshooting_hints"`
	ServerCount          int                  `json:"server_count"`
	ToolCount            int                  `json:"tool_count"`
	PromptCount          int                  `json:"prompt_count"`
	ResourceCount        int                  `json:"resource_count"`
	EmptyState           *DashboardEmptyState `json:"empty_state,omitempty"`
}
