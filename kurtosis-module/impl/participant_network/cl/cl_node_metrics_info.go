package cl

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
