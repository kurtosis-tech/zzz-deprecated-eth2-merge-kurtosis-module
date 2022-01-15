package cl

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"path"
)

type NodeType string

const (
	rawKeysDirname = "keys"
	rawSecretsDirname = "secrets"

	lodestarSecretsDirname = "lodestar-secrets"

	nimbusKeysDirname = "nimbus-keys"
	prysmDirname = "prysm"

	tekuKeysDirname = "teku-keys"
	tekuSecretsDirname = "teku-secrets"

)

type NodeTypeKeystoreDirpaths struct {
	// TODO can add pubkeys.json if needed

	RawKeysDirpath    string
	RawSecretsDirpath string

	LodestarSecretsDirpath string

	NimbusKeysDirpath string

	PrysmDirpath string

	TekuKeysDirpath    string
	TekuSecretsDirpath string
}
func NewNodeTypeKeystoreDirpathsFromOutputSharedPath(sharedPath *services.SharedPath) *NodeTypeKeystoreDirpaths {
	outputDirpathOnModuleContainer := sharedPath.GetAbsPathOnThisContainer()
	return &NodeTypeKeystoreDirpaths{
		RawKeysDirpath:         path.Join(outputDirpathOnModuleContainer, rawKeysDirname),
		RawSecretsDirpath:      path.Join(outputDirpathOnModuleContainer, rawSecretsDirname),
		LodestarSecretsDirpath: path.Join(outputDirpathOnModuleContainer, lodestarSecretsDirname),
		NimbusKeysDirpath:      path.Join(outputDirpathOnModuleContainer, nimbusKeysDirname),
		PrysmDirpath:           path.Join(outputDirpathOnModuleContainer, prysmDirname),
		TekuKeysDirpath:        path.Join(outputDirpathOnModuleContainer, tekuKeysDirname),
		TekuSecretsDirpath:     path.Join(outputDirpathOnModuleContainer, tekuSecretsDirname),
	}
}


// Package object containing information about the keystores that were generated for validators
//  during genesis creation
type GenerateKeystoresResult struct {
	// Prysm keystores are encrypted with a password; this is the password
	PrysmPassword string

	// Contains keystores-per-client-type for each node in the network
	PerNodeKeystoreDirpaths []*NodeTypeKeystoreDirpaths
}

func NewGenerateKeystoresResult(prysmPassword string, perNodeKeystoreDirpaths []*NodeTypeKeystoreDirpaths) *GenerateKeystoresResult {
	return &GenerateKeystoresResult{PrysmPassword: prysmPassword, PerNodeKeystoreDirpaths: perNodeKeystoreDirpaths}
}
