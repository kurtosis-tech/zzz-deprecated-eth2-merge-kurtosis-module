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

// The struct has unexported fields as they start with small letters.
// These fields don't get Marshaled by default as they are hidden.
// We define this custom MarshalJSON function so that we can convert this struct to JSON properly.
// We don't use a pointer receiver as someone might marshal a pointer or a value itself.
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
