package el_genesis

// Represents the paths to the EL genesis files *on the module container*
type ELGenesisData struct {
	// Path to the directory holding all the EL genesis files
	parentDirpath string
	
	chainspecJsonFilepath string

	gethGenesisJsonFilepath string

	nethermindGenesisJsonFilepath string
}

func newELGenesisData(parentDirpath string, chainspecJsonFilepath string, gethGenesisJsonFilepath string, nethermindGenesisJsonFilepath string) *ELGenesisData {
	return &ELGenesisData{parentDirpath: parentDirpath, chainspecJsonFilepath: chainspecJsonFilepath, gethGenesisJsonFilepath: gethGenesisJsonFilepath, nethermindGenesisJsonFilepath: nethermindGenesisJsonFilepath}
}

func (paths *ELGenesisData) GetParentDirpath() string {
	return paths.parentDirpath
}
func (paths *ELGenesisData) GetChainspecJsonFilepath() string {
	return paths.chainspecJsonFilepath
}
func (paths *ELGenesisData) GetGethGenesisJsonFilepath() string {
	return paths.gethGenesisJsonFilepath
}
func (paths *ELGenesisData) GetNethermindGenesisJsonFilepath() string {
	return paths.nethermindGenesisJsonFilepath
}
