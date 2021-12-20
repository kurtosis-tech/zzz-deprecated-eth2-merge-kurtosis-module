package ethereum_genesis_generator

type CLGenesisPaths struct {
	// Path to the directory holding all the CL genesis files
	parentDirpath string

	configYmlFilepath string

	genesisSszFilepath string
}

func NewCLGenesisPaths(parentDirpath string, configYmlFilepath string, genesisSszFilepath string) *CLGenesisPaths {
	return &CLGenesisPaths{parentDirpath: parentDirpath, configYmlFilepath: configYmlFilepath, genesisSszFilepath: genesisSszFilepath}
}

func (paths *CLGenesisPaths) GetParentDirpath() string {
	return paths.parentDirpath
}
func (paths *CLGenesisPaths) GetConfigYMLFilepath() string {
	return paths.configYmlFilepath
}
func (paths *CLGenesisPaths) GetGenesisSSZFilepath() string {
	return paths.genesisSszFilepath
}
