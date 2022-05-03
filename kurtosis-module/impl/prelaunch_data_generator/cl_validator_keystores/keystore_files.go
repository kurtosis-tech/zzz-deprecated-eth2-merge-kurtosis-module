package cl_validator_keystores

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

// One of these will be created per node we're trying to start
type KeystoreFiles struct {
	FilesArtifactID services.FilesArtifactID

	// ------------ All directories below are relative to the root of the files artifact -----------------
	RawKeysRelativeDirpath    string
	RawSecretsRelativeDirpath string

	LodestarSecretsRelativeDirpath string

	NimbusKeysRelativeDirpath string

	PrysmRelativeDirpath string

	TekuKeysRelativeDirpath    string
	TekuSecretsRelativeDirpath string
}

func NewKeystoreFiles(filesArtifactID services.FilesArtifactID, rawKeysRelativeDirpath string, rawSecretsRelativeDirpath string, lodestarSecretsRelativeDirpath string, nimbusKeysRelativeDirpath string, prysmRelativeDirpath string, tekuKeysRelativeDirpath string, tekuSecretsRelativeDirpath string) *KeystoreFiles {
	return &KeystoreFiles{FilesArtifactID: filesArtifactID, RawKeysRelativeDirpath: rawKeysRelativeDirpath, RawSecretsRelativeDirpath: rawSecretsRelativeDirpath, LodestarSecretsRelativeDirpath: lodestarSecretsRelativeDirpath, NimbusKeysRelativeDirpath: nimbusKeysRelativeDirpath, PrysmRelativeDirpath: prysmRelativeDirpath, TekuKeysRelativeDirpath: tekuKeysRelativeDirpath, TekuSecretsRelativeDirpath: tekuSecretsRelativeDirpath}
}
