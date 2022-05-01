package el_genesis

import "github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"

// Represents the paths to the EL genesis files *on the module container*
type ELGenesisData struct {
	// The ID of the files artifact containing EL genesis information
	filesArtifactId services.FilesArtifactID

	// Relative filepaths inside the files artifact where various files can be found
	jwtSecretRelativeFilepath string
	gethGenesisJsonRelativeFilepath string
	nethermindGenesisJsonRelativeFilepath string
	besuGenesisJsonRelativeFilepath string
}

func newELGenesisData(filesArtifactId services.FilesArtifactID, jwtSecretRelativeFilepath string, gethGenesisJsonRelativeFilepath string, nethermindGenesisJsonRelativeFilepath string, besuGenesisJsonRelativeFilepath string) *ELGenesisData {
	return &ELGenesisData{filesArtifactId: filesArtifactId, jwtSecretRelativeFilepath: jwtSecretRelativeFilepath, gethGenesisJsonRelativeFilepath: gethGenesisJsonRelativeFilepath, nethermindGenesisJsonRelativeFilepath: nethermindGenesisJsonRelativeFilepath, besuGenesisJsonRelativeFilepath: besuGenesisJsonRelativeFilepath}
}

func (data *ELGenesisData) GetFilesArtifactID() services.FilesArtifactID {
	return data.filesArtifactId
}
func (data *ELGenesisData) GetJWTSecretRelativeFilepath() string {
	return data.jwtSecretRelativeFilepath
}
func (data *ELGenesisData) GetGethGenesisJsonRelativeFilepath() string {
	return data.gethGenesisJsonRelativeFilepath
}
func (data *ELGenesisData) GetNethermindGenesisJsonRelativeFilepath() string {
	return data.nethermindGenesisJsonRelativeFilepath
}
func (data *ELGenesisData) GetBesuGenesisJsonRelativeFilepath() string {
	return data.besuGenesisJsonRelativeFilepath
}
