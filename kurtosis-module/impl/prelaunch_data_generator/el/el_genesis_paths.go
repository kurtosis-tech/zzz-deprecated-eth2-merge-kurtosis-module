package el

// Represents the paths to the EL genesis files *on the module container*
type ELGenesisPaths struct {
	// Path to the directory holding all the EL genesis files
	parentDirpath string

	gethGenesisJsonFilepath string

	nethermindGenesisJsonFilepath string
}

func NewELGenesisPaths(parentDirpath string, gethGenesisJsonFilepath string, nethermindGenesisJsonFilepath string) *ELGenesisPaths {
	return &ELGenesisPaths{parentDirpath: parentDirpath, gethGenesisJsonFilepath: gethGenesisJsonFilepath, nethermindGenesisJsonFilepath: nethermindGenesisJsonFilepath}
}

func (paths *ELGenesisPaths) GetParentDirpath() string {
	return paths.parentDirpath
}
func (paths *ELGenesisPaths) GetGethGenesisJsonFilepath() string {
	return paths.gethGenesisJsonFilepath
}
func (paths *ELGenesisPaths) GetNethermindGenesisJsonFilepath() string {
	return paths.nethermindGenesisJsonFilepath
}
