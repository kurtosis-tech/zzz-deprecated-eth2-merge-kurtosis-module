package cl_validator_keystores

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

// Package object containing information about the keystores that were generated for validators
//  during genesis creation
type GenerateKeystoresResult struct {
	// Files artifact ID where the Prysm password is stored
	PrysmPasswordArtifactId services.FilesArtifactID

	// Relative to root of files artifact
	PrysmPasswordRelativeFilepath string

	// Contains keystores-per-client-type for each node in the network
	PerNodeKeystores []*KeystoreFiles
}

func NewGenerateKeystoresResult(prysmPasswordArtifactId services.FilesArtifactID, prysmPasswordRelativeFilepath string, perNodeKeystores []*KeystoreFiles) *GenerateKeystoresResult {
	return &GenerateKeystoresResult{PrysmPasswordArtifactId: prysmPasswordArtifactId, PrysmPasswordRelativeFilepath: prysmPasswordRelativeFilepath, PerNodeKeystores: perNodeKeystores}
}
