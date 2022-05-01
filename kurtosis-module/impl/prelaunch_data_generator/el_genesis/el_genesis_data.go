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

func (paths *ELGenesisData) GetJWTSecretRelativeFilepath() string {
	return paths.jwtSecretRelativeFilepath
}
func (paths *ELGenesisData) GetGethGenesisJsonRelativeFilepath() string {
	return paths.gethGenesisJsonRelativeFilepath
}
func (paths *ELGenesisData) GetNethermindGenesisJsonRelativeFilepath() string {
	return paths.nethermindGenesisJsonRelativeFilepath
}
func (paths *ELGenesisData) GetBesuGenesisJsonRelativeFilepath() string {
	return paths.besuGenesisJsonRelativeFilepath
}
