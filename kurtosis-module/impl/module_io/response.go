package module_io

// The structure that will be returned, JSON-serialized, from calling this module
type ExecuteResponse struct {
	GrafanaInfo *GrafanaInfo `json:"grafana"`
}

type GrafanaInfo struct {
	DashboardPath string `json:"dashboardPath"`
	User          string `json:"user"`
	Password      string `json:"password"`
}
