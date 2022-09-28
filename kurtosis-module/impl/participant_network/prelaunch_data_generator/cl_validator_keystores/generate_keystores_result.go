package cl_validator_keystores

import "github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"

// Package object containing information about the keystores that were generated for validators
//  during genesis creation
type GenerateKeystoresResult struct {
	// Files artifact UUID where the Prysm password is stored
	PrysmPasswordArtifactUUid services.FilesArtifactUUID

	// Relative to root of files artifact
	PrysmPasswordRelativeFilepath string

	// Contains keystores-per-client-type for each node in the network
	PerNodeKeystores []*KeystoreFiles
}

func NewGenerateKeystoresResult(prysmPasswordArtifactUuid services.FilesArtifactUUID, prysmPasswordRelativeFilepath string, perNodeKeystores []*KeystoreFiles) *GenerateKeystoresResult {
	return &GenerateKeystoresResult{PrysmPasswordArtifactUUid: prysmPasswordArtifactUuid, PrysmPasswordRelativeFilepath: prysmPasswordRelativeFilepath, PerNodeKeystores: perNodeKeystores}
}
