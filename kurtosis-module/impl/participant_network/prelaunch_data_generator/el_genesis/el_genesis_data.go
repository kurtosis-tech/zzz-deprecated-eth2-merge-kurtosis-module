package el_genesis

import "github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"

// Represents the paths to the EL genesis files *on the module container*
type ELGenesisData struct {
	// The UUID of the files artifact containing EL genesis information
	filesArtifactUuid services.FilesArtifactUUID

	// Relative filepaths inside the files artifact where various files can be found
	jwtSecretRelativeFilepath string
	gethGenesisJsonRelativeFilepath string
	erigonGenesisJsonRelativeFilepath string
	nethermindGenesisJsonRelativeFilepath string
	besuGenesisJsonRelativeFilepath string
}

func newELGenesisData(filesArtifactUuid services.FilesArtifactUUID, jwtSecretRelativeFilepath string, gethGenesisJsonRelativeFilepath string, erigonGenesisJsonRelativeFilepath string, nethermindGenesisJsonRelativeFilepath string, besuGenesisJsonRelativeFilepath string) *ELGenesisData {
	return &ELGenesisData{filesArtifactUuid: filesArtifactUuid, jwtSecretRelativeFilepath: jwtSecretRelativeFilepath, gethGenesisJsonRelativeFilepath: gethGenesisJsonRelativeFilepath, erigonGenesisJsonRelativeFilepath: erigonGenesisJsonRelativeFilepath, nethermindGenesisJsonRelativeFilepath: nethermindGenesisJsonRelativeFilepath, besuGenesisJsonRelativeFilepath: besuGenesisJsonRelativeFilepath}
}

func (data *ELGenesisData) GetFilesArtifactUUID() services.FilesArtifactUUID {
	return data.filesArtifactUuid
}
func (data *ELGenesisData) GetJWTSecretRelativeFilepath() string {
	return data.jwtSecretRelativeFilepath
}
func (data *ELGenesisData) GetGethGenesisJsonRelativeFilepath() string {
	return data.gethGenesisJsonRelativeFilepath
}
func (data *ELGenesisData) GetErigonGenesisJsonRelativeFilepath() string {
	return data.erigonGenesisJsonRelativeFilepath
}
func (data *ELGenesisData) GetNethermindGenesisJsonRelativeFilepath() string {
	return data.nethermindGenesisJsonRelativeFilepath
}
func (data *ELGenesisData) GetBesuGenesisJsonRelativeFilepath() string {
	return data.besuGenesisJsonRelativeFilepath
}
