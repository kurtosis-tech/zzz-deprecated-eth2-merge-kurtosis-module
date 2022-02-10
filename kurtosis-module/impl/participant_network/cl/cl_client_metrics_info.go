package cl

type CLClientMetricsInfo struct {
	grafanaDashboardConfigFilename string
	clNodesMetricsInfo             []*CLNodeMetricsInfo
}

func NewCLMetricsInfo(grafanaDashboardConfigFilename string, clNodesMetricsInfo []*CLNodeMetricsInfo) *CLClientMetricsInfo {
	return &CLClientMetricsInfo{grafanaDashboardConfigFilename: grafanaDashboardConfigFilename, clNodesMetricsInfo: clNodesMetricsInfo}
}

func (clMetricsInfo *CLClientMetricsInfo) GetGrafanaDashboardConfigFilename() string {
	return clMetricsInfo.grafanaDashboardConfigFilename
}

func (clMetricsInfo *CLClientMetricsInfo) GetClNodesMetricsInfo() []*CLNodeMetricsInfo {
	return clMetricsInfo.clNodesMetricsInfo
}


type CLNodeMetricsInfo struct {
	name string
	path string
	url string
}

func NewCLNodeMetricsInfo(name string, path string, url string) *CLNodeMetricsInfo {
	return &CLNodeMetricsInfo{name: name, path: path, url: url}
}

func (clNodeMetricInfo *CLNodeMetricsInfo) GetName() string {
	return clNodeMetricInfo.name
}

func (clNodeMetricInfo *CLNodeMetricsInfo) GetPath() string {
	return clNodeMetricInfo.path
}

func (clNodeMetricInfo *CLNodeMetricsInfo) GetURL() string {
	return clNodeMetricInfo.url
}
