package cl_validator_keystores

// Package object containing information about the keystores that were generated for validators
//  during genesis creation
type GenerateKeystoresResult struct {
	// Prysm keystores are encrypted with a password; this is the password
	PrysmPassword string

	// Contains keystores-per-client-type for each node in the network
	PerNodeKeystores []*KeystoreFiles
}

func NewGenerateKeystoresResult(prysmPassword string, perNodeKeystores []*KeystoreFiles) *GenerateKeystoresResult {
	return &GenerateKeystoresResult{PrysmPassword: prysmPassword, PerNodeKeystores: perNodeKeystores}
}
