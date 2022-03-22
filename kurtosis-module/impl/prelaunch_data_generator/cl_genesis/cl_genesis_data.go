package cl_genesis

type CLGenesisData struct {
	// Path to the directory holding all the CL genesis files
	parentDirpath string
	
	jwtSecretFilepath string

	configYmlFilepath string

	genesisSszFilepath string
}

func newCLGenesisData(parentDirpath string, jwtSecretFilepath string, configYmlFilepath string, genesisSszFilepath string) *CLGenesisData {
	return &CLGenesisData{parentDirpath: parentDirpath, jwtSecretFilepath: jwtSecretFilepath, configYmlFilepath: configYmlFilepath, genesisSszFilepath: genesisSszFilepath}
}

func (paths *CLGenesisData) GetParentDirpath() string {
	return paths.parentDirpath
}
func (paths *CLGenesisData) GetJWTSecretFilepath() string {
	return paths.jwtSecretFilepath
}
func (paths *CLGenesisData) GetConfigYMLFilepath() string {
	return paths.configYmlFilepath
}
func (paths *CLGenesisData) GetGenesisSSZFilepath() string {
	return paths.genesisSszFilepath
}
