package module_io

// The structure that will be returned, JSON-serialized, from calling this module
type ExecuteResponse struct {
	ForkmonPublicURL string	`json:"forkmonUrl"`
}
