package el

// Represents the paths to the EL genesis files *on the module container*
type ELPrelaunchData struct {
	// Path to the directory holding all the EL genesis files
	parentDirpath string
	
	chainspecJsonFilepath string

	gethGenesisJsonFilepath string

	nethermindGenesisJsonFilepath string
}

func NewELPrelaunchData(parentDirpath string, chainspecJsonFilepath string, gethGenesisJsonFilepath string, nethermindGenesisJsonFilepath string) *ELPrelaunchData {
	return &ELPrelaunchData{parentDirpath: parentDirpath, chainspecJsonFilepath: chainspecJsonFilepath, gethGenesisJsonFilepath: gethGenesisJsonFilepath, nethermindGenesisJsonFilepath: nethermindGenesisJsonFilepath}
}

func (paths *ELPrelaunchData) GetParentDirpath() string {
	return paths.parentDirpath
}
func (paths *ELPrelaunchData) GetChainspecJsonFilepath() string {
	return paths.chainspecJsonFilepath
}
func (paths *ELPrelaunchData) GetGethGenesisJsonFilepath() string {
	return paths.gethGenesisJsonFilepath
}
func (paths *ELPrelaunchData) GetNethermindGenesisJsonFilepath() string {
	return paths.nethermindGenesisJsonFilepath
}
