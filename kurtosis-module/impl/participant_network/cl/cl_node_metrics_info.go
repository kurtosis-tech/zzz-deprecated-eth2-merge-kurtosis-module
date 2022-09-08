package cl

import "encoding/json"

type CLNodeMetricsInfo struct {
	name string
	path string
	url  string
}

func NewCLNodeMetricsInfo(name string, path string, url string) *CLNodeMetricsInfo {
	return &CLNodeMetricsInfo{name: name, path: path, url: url}
}

func (cLNodeMetricsInfo CLNodeMetricsInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Url  string `json:"url"`
	}{
		Name: cLNodeMetricsInfo.name,
		Path: cLNodeMetricsInfo.path,
		Url:  cLNodeMetricsInfo.url,
	})
}
