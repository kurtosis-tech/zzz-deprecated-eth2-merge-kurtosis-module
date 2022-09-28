package cl_validator_keystores

import "github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"

// One of these will be created per node we're trying to start
type KeystoreFiles struct {
	FilesArtifactUUID services.FilesArtifactUUID

	// ------------ All directories below are relative to the root of the files artifact -----------------
	RawKeysRelativeDirpath    string
	RawSecretsRelativeDirpath string

	LodestarSecretsRelativeDirpath string

	NimbusKeysRelativeDirpath string

	PrysmRelativeDirpath string

	TekuKeysRelativeDirpath    string
	TekuSecretsRelativeDirpath string
}

func NewKeystoreFiles(filesArtifactUUID services.FilesArtifactUUID, rawKeysRelativeDirpath string, rawSecretsRelativeDirpath string, lodestarSecretsRelativeDirpath string, nimbusKeysRelativeDirpath string, prysmRelativeDirpath string, tekuKeysRelativeDirpath string, tekuSecretsRelativeDirpath string) *KeystoreFiles {
	return &KeystoreFiles{FilesArtifactUUID: filesArtifactUUID, RawKeysRelativeDirpath: rawKeysRelativeDirpath, RawSecretsRelativeDirpath: rawSecretsRelativeDirpath, LodestarSecretsRelativeDirpath: lodestarSecretsRelativeDirpath, NimbusKeysRelativeDirpath: nimbusKeysRelativeDirpath, PrysmRelativeDirpath: prysmRelativeDirpath, TekuKeysRelativeDirpath: tekuKeysRelativeDirpath, TekuSecretsRelativeDirpath: tekuSecretsRelativeDirpath}
}
