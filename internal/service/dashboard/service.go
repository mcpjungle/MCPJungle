package dashboard

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
	"github.com/mcpjungle/mcpjungle/pkg/types"
	"github.com/mcpjungle/mcpjungle/pkg/version"
	"gorm.io/gorm"
)

type Service struct {
	db             *gorm.DB
	metricsEnabled bool
}

func NewService(db *gorm.DB, metricsEnabled bool) *Service {
	return &Service{db: db, metricsEnabled: metricsEnabled}
}

type serverInventory struct {
	model.McpServer
	ToolCount      int
	PromptCount    int
	ResourceCount  int
	LastEntitySeen time.Time
}

func (s *Service) Overview(mode model.ServerMode, baseURL string) (*types.DashboardOverviewResponse, error) {
	inventory, err := s.loadServerInventory()
	if err != nil {
		return nil, err
	}

	toolCount, promptCount, resourceCount, err := s.loadEntityCounts()
	if err != nil {
		return nil, err
	}

	status := types.DashboardStatusRunning
	troubleshooting := collectTroubleshootingHints(inventory, toolCount, promptCount, resourceCount)
	if len(inventory) > 0 && hasDiscoveryGap(inventory) {
		status = types.DashboardStatusDegraded
	}

	resp := &types.DashboardOverviewResponse{
		Status:          status,
		Mode:            string(mode),
		Version:         version.GetVersion(),
		Endpoints:       buildEndpoints(baseURL),
		ServerCount:     len(inventory),
		ToolCount:       toolCount,
		PromptCount:     promptCount,
		ResourceCount:   resourceCount,
		Troubleshooting: troubleshooting,
	}
	if len(inventory) == 0 {
		resp.EmptyState = noServersEmptyState()
	}

	return resp, nil
}

func (s *Service) Servers() (*types.DashboardServersResponse, error) {
	inventory, err := s.loadServerInventory()
	if err != nil {
		return nil, err
	}

	resp := &types.DashboardServersResponse{
		Servers: make([]types.DashboardServer, 0, len(inventory)),
	}
	for _, inv := range inventory {
		summary := summarizeServerConfig(inv.McpServer)
		resp.Servers = append(resp.Servers, types.DashboardServer{
			Name:              inv.Name,
			Transport:         string(inv.Transport),
			Status:            deriveServerStatus(inv),
			ToolCount:         inv.ToolCount,
			PromptCount:       inv.PromptCount,
			ResourceCount:     inv.ResourceCount,
			LastDiscoveredAt:  formatTime(inv.LastEntitySeen),
			UpdatedAt:         formatTime(inv.UpdatedAt),
			ConnectionSummary: summary.SanitizedSummary,
			ConfigSummary:     summary,
		})
	}

	if len(resp.Servers) == 0 {
		resp.EmptyState = noServersEmptyState()
	}

	return resp, nil
}

func (s *Service) Tools() (*types.DashboardToolsResponse, error) {
	var tools []model.Tool
	if err := s.db.Preload("Server").Order("name asc").Find(&tools).Error; err != nil {
		return nil, err
	}

	resp := &types.DashboardToolsResponse{
		Tools: make([]types.DashboardTool, 0, len(tools)),
	}
	for _, tool := range tools {
		canonicalName := mergeServerName(tool.Server.Name, tool.Name, "__")
		resp.Tools = append(resp.Tools, types.DashboardTool{
			Name:           tool.Name,
			CanonicalName:  canonicalName,
			Server:         tool.Server.Name,
			Description:    tool.Description,
			Enabled:        tool.Enabled,
			InputSchema:    decodeJSONMap(tool.InputSchema),
			InputPreview:   compactJSON(tool.InputSchema),
			Transport:      string(tool.Server.Transport),
			ServerStatus:   string(deriveServerStatusFromCounts(tool.Server.Transport, 1, 0, 0)),
			AnnotationKeys: sortedKeys(decodeJSONMap(tool.Annotations)),
		})
	}
	if len(resp.Tools) == 0 {
		resp.EmptyState = emptyState(
			"No tools discovered yet",
			"MCPJungle is running, but it has not discovered any tools from registered servers yet.",
			[]string{"mcpjungle list", "mcpjungle get <server-name>"},
		)
	}
	return resp, nil
}

func (s *Service) Prompts() (*types.DashboardPromptsResponse, error) {
	var prompts []model.Prompt
	if err := s.db.Preload("Server").Order("name asc").Find(&prompts).Error; err != nil {
		return nil, err
	}

	resp := &types.DashboardPromptsResponse{
		Prompts: make([]types.DashboardPrompt, 0, len(prompts)),
	}
	for _, prompt := range prompts {
		arguments := decodeJSONArray(prompt.Arguments)
		resp.Prompts = append(resp.Prompts, types.DashboardPrompt{
			Name:             prompt.Name,
			CanonicalName:    mergeServerName(prompt.Server.Name, prompt.Name, "__"),
			Server:           prompt.Server.Name,
			Description:      prompt.Description,
			Enabled:          prompt.Enabled,
			Arguments:        arguments,
			ArgumentsPreview: compactJSONArray(arguments),
			Transport:        string(prompt.Server.Transport),
			ServerStatus:     string(deriveServerStatusFromCounts(prompt.Server.Transport, 0, 1, 0)),
		})
	}
	if len(resp.Prompts) == 0 {
		resp.EmptyState = emptyState(
			"No prompts discovered yet",
			"Registered servers can expose prompt templates. None are currently available.",
			[]string{"mcpjungle list", "mcpjungle get <server-name>"},
		)
	}
	return resp, nil
}

func (s *Service) Resources() (*types.DashboardResourcesResponse, error) {
	var resources []model.Resource
	if err := s.db.Preload("Server").Order("name asc").Find(&resources).Error; err != nil {
		return nil, err
	}

	resp := &types.DashboardResourcesResponse{
		Resources: make([]types.DashboardResource, 0, len(resources)),
	}
	for _, resource := range resources {
		resp.Resources = append(resp.Resources, types.DashboardResource{
			URI:          resource.URI,
			Name:         resource.Name,
			Server:       resource.Server.Name,
			Description:  resource.Description,
			MIMEType:     resource.MIMEType,
			Enabled:      resource.Enabled,
			Transport:    string(resource.Server.Transport),
			ServerStatus: string(deriveServerStatusFromCounts(resource.Server.Transport, 0, 0, 1)),
		})
	}
	if len(resp.Resources) == 0 {
		resp.EmptyState = emptyState(
			"No resources discovered yet",
			"Registered servers can expose MCP resources. None are currently available.",
			[]string{"mcpjungle list", "mcpjungle get <server-name>"},
		)
	}
	return resp, nil
}

func (s *Service) Diagnostics(mode model.ServerMode, baseURL string) (*types.DashboardDiagnosticsResponse, error) {
	inventory, err := s.loadServerInventory()
	if err != nil {
		return nil, err
	}

	toolCount, promptCount, resourceCount, err := s.loadEntityCounts()
	if err != nil {
		return nil, err
	}

	resp := &types.DashboardDiagnosticsResponse{
		Version:              version.GetVersion(),
		Mode:                 string(mode),
		ConfigSource:         "database (server_config table)",
		Database:             s.db.Dialector.Name(),
		EnabledTransports:    enabledTransports(inventory),
		PrimaryEndpoint:      strings.TrimRight(baseURL, "/") + "/mcp",
		TroubleshootingHints: collectTroubleshootingHints(inventory, toolCount, promptCount, resourceCount),
		ServerCount:          len(inventory),
		ToolCount:            toolCount,
		PromptCount:          promptCount,
		ResourceCount:        resourceCount,
	}
	if s.metricsEnabled {
		resp.MetricsEndpoint = strings.TrimRight(baseURL, "/") + "/metrics"
	}
	if len(inventory) == 0 {
		resp.EmptyState = noServersEmptyState()
	}
	return resp, nil
}

func (s *Service) loadServerInventory() ([]serverInventory, error) {
	var servers []model.McpServer
	if err := s.db.Order("name asc").Find(&servers).Error; err != nil {
		return nil, err
	}

	toolCounts, err := groupedCounts[model.Tool](s.db)
	if err != nil {
		return nil, err
	}
	promptCounts, err := groupedCounts[model.Prompt](s.db)
	if err != nil {
		return nil, err
	}
	resourceCounts, err := groupedCounts[model.Resource](s.db)
	if err != nil {
		return nil, err
	}

	inventory := make([]serverInventory, 0, len(servers))
	for _, server := range servers {
		inventory = append(inventory, serverInventory{
			McpServer:      server,
			ToolCount:      toolCounts[server.ID],
			PromptCount:    promptCounts[server.ID],
			ResourceCount:  resourceCounts[server.ID],
			LastEntitySeen: server.UpdatedAt,
		})
	}
	return inventory, nil
}

func (s *Service) loadEntityCounts() (int, int, int, error) {
	var toolCount int64
	if err := s.db.Model(&model.Tool{}).Count(&toolCount).Error; err != nil {
		return 0, 0, 0, err
	}
	var promptCount int64
	if err := s.db.Model(&model.Prompt{}).Count(&promptCount).Error; err != nil {
		return 0, 0, 0, err
	}
	var resourceCount int64
	if err := s.db.Model(&model.Resource{}).Count(&resourceCount).Error; err != nil {
		return 0, 0, 0, err
	}
	return int(toolCount), int(promptCount), int(resourceCount), nil
}

type groupedCountRow struct {
	ServerID uint
	Count    int
}

func groupedCounts[T any](db *gorm.DB) (map[uint]int, error) {
	var rows []groupedCountRow
	if err := db.Model(new(T)).
		Select("server_id, COUNT(*) AS count").
		Group("server_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[uint]int, len(rows))
	for _, row := range rows {
		counts[row.ServerID] = row.Count
	}
	return counts, nil
}

func deriveServerStatus(inv serverInventory) types.DashboardServerStatus {
	return deriveServerStatusFromCounts(inv.Transport, inv.ToolCount, inv.PromptCount, inv.ResourceCount)
}

func deriveServerStatusFromCounts(transport types.McpServerTransport, toolCount, promptCount, resourceCount int) types.DashboardServerStatus {
	total := toolCount + promptCount + resourceCount
	if total == 0 {
		return types.DashboardServerStatusUnknown
	}
	if transport == types.TransportStdio {
		return types.DashboardServerStatusConnected
	}
	return types.DashboardServerStatusReachable
}

func summarizeServerConfig(server model.McpServer) types.DashboardServerConfigSummary {
	summary := types.DashboardServerConfigSummary{
		Kind:        string(server.Transport),
		SessionMode: string(server.SessionMode),
		Description: server.Description,
	}

	switch server.Transport {
	case types.TransportStreamableHTTP:
		if conf, err := server.GetStreamableHTTPConfig(); err == nil {
			summary.Target = sanitizeURL(conf.URL)
			summary.HeaderKeys = sortedHeaderKeys(conf.Headers)
			summary.SanitizedSummary = fmt.Sprintf(
				"%s via streamable HTTP%s",
				summary.Target,
				formatHeaderKeySuffix(summary.HeaderKeys),
			)
		}
	case types.TransportSSE:
		if conf, err := server.GetSSEConfig(); err == nil {
			summary.Target = sanitizeURL(conf.URL)
			summary.SanitizedSummary = fmt.Sprintf("%s via SSE", summary.Target)
		}
	case types.TransportStdio:
		if conf, err := server.GetStdioConfig(); err == nil {
			summary.Command = conf.Command
			summary.ArgumentCount = len(conf.Args)
			summary.EnvKeys = sortedKeysString(conf.Env)
			summary.SanitizedSummary = fmt.Sprintf(
				"%s (%d args%s)",
				conf.Command,
				len(conf.Args),
				formatEnvKeySuffix(summary.EnvKeys),
			)
		}
	}

	if summary.SanitizedSummary == "" {
		summary.SanitizedSummary = "Configuration summary unavailable"
	}

	return summary
}

func sanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func buildEndpoints(baseURL string) []types.DashboardEndpoint {
	root := strings.TrimRight(baseURL, "/")
	return []types.DashboardEndpoint{
		{Label: "Primary MCP endpoint", URL: root + "/mcp"},
		{Label: "SSE endpoint", URL: root + "/sse"},
	}
}

func collectTroubleshootingHints(inventory []serverInventory, toolCount, promptCount, resourceCount int) []string {
	hints := []string{}
	if len(inventory) == 0 {
		hints = append(hints, "No servers registered yet")
	}
	if len(inventory) > 0 && toolCount == 0 {
		hints = append(hints, "Server registered but no tools discovered")
	}
	if len(inventory) > 0 && (promptCount == 0 || resourceCount == 0) {
		hints = append(hints, "Prompt/resource discovery failed")
	}
	hints = append(hints, "Check CLI logs for detailed errors")
	return hints
}

func hasDiscoveryGap(inventory []serverInventory) bool {
	for _, inv := range inventory {
		if inv.ToolCount == 0 && inv.PromptCount == 0 && inv.ResourceCount == 0 {
			return true
		}
	}
	return false
}

func noServersEmptyState() *types.DashboardEmptyState {
	return emptyState(
		"No servers registered yet",
		"Register an MCP server from the CLI, then refresh the dashboard to inspect tools, prompts, and resources.",
		[]string{
			"mcpjungle register --transport stdio --name filesystem --command npx --args -y,@modelcontextprotocol/server-filesystem",
			"mcpjungle list",
		},
	)
}

func emptyState(title, description string, commands []string) *types.DashboardEmptyState {
	return &types.DashboardEmptyState{
		Title:       title,
		Description: description,
		Commands:    commands,
	}
}

func enabledTransports(inventory []serverInventory) []string {
	set := map[string]struct{}{
		string(types.TransportStreamableHTTP): {},
		string(types.TransportSSE):            {},
		string(types.TransportStdio):          {},
	}
	for _, server := range inventory {
		set[string(server.Transport)] = struct{}{}
	}

	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func decodeJSONArray(raw []byte) []map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var value []map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func compactJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return truncateString(string(encoded), 160)
}

func compactJSONArray(value []map[string]any) string {
	if len(value) == 0 {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return truncateString(string(encoded), 160)
}

func truncateString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max-1] + "…"
}

func sortedKeys(value map[string]any) []string {
	if len(value) == 0 {
		return nil
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysString(value map[string]string) []string {
	if len(value) == 0 {
		return nil
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedHeaderKeys(value map[string]string) []string {
	keys := sortedKeysString(value)
	filtered := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.EqualFold(key, "authorization") {
			continue
		}
		filtered = append(filtered, key)
	}
	return filtered
}

func formatHeaderKeySuffix(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return fmt.Sprintf(" with %d custom headers", len(keys))
}

func formatEnvKeySuffix(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return fmt.Sprintf(", %d env keys", len(keys))
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func mergeServerName(serverName, itemName, separator string) string {
	if serverName == "" {
		return itemName
	}
	return serverName + separator + itemName
}
