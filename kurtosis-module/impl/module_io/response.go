package module_io

// The structure that will be returned, JSON-serialized, from calling this module
type ExecuteResponse struct {
	ForkmonPublicURL string	`json:"forkmonUrl"`
	PrometheusPublicURL string `json:"prometheusUrl"`
	GrafanaInfo *GrafanaInfo `json:"grafana"`
	ClPeers []string `json:"clPeers"`
	ElPeers []string `json:"elPeers"`
}

type GrafanaInfo struct {
	PublicURL string `json:"url"`
	DashboardURL string `json:"dashboardUrl"`
	User string `json:"user"`
	Password string `json:"password"`
}