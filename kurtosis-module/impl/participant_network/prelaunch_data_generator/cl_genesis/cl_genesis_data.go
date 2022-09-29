package cl_genesis

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type CLGenesisData struct {
	filesArtifactUuid services.FilesArtifactUUID

	// Various filepaths, relative to the root of the files artifact
	jwtSecretRelativeFilepath  string
	configYmlRelativeFilepath  string
	genesisSszRelativeFilepath string
}

func newCLGenesisData(filesArtifactUuid services.FilesArtifactUUID, jwtSecretRelativeFilepath string, configYmlRelativeFilepath string, genesisSszRelativeFilepath string) *CLGenesisData {
	return &CLGenesisData{filesArtifactUuid: filesArtifactUuid, jwtSecretRelativeFilepath: jwtSecretRelativeFilepath, configYmlRelativeFilepath: configYmlRelativeFilepath, genesisSszRelativeFilepath: genesisSszRelativeFilepath}
}

func (paths *CLGenesisData) GetFilesArtifactUUID() services.FilesArtifactUUID {
	return paths.filesArtifactUuid
}
func (paths *CLGenesisData) GetJWTSecretRelativeFilepath() string {
	return paths.jwtSecretRelativeFilepath
}
func (paths *CLGenesisData) GetConfigYMLRelativeFilepath() string {
	return paths.configYmlRelativeFilepath
}
func (paths *CLGenesisData) GetGenesisSSZRelativeFilepath() string {
	return paths.genesisSszRelativeFilepath
}
