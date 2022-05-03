package cl_genesis

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

type CLGenesisData struct {
	filesArtifactId services.FilesArtifactID

	// Various filepaths, relative to the root of the files artifact
	jwtSecretRelativeFilepath string
	configYmlRelativeFilepath string
	genesisSszRelativeFilepath string
}

func newCLGenesisData(filesArtifactId services.FilesArtifactID, jwtSecretRelativeFilepath string, configYmlRelativeFilepath string, genesisSszRelativeFilepath string) *CLGenesisData {
	return &CLGenesisData{filesArtifactId: filesArtifactId, jwtSecretRelativeFilepath: jwtSecretRelativeFilepath, configYmlRelativeFilepath: configYmlRelativeFilepath, genesisSszRelativeFilepath: genesisSszRelativeFilepath}
}

func (paths *CLGenesisData) GetFilesArtifactID() services.FilesArtifactID {
	return paths.filesArtifactId
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
