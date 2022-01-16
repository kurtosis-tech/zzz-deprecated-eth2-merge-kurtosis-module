package cl_genesis

type CLGenesisData struct {
	genesisUnixTimestamp uint64

	// Path to the directory holding all the CL genesis files
	parentDirpath string

	configYmlFilepath string

	genesisSszFilepath string
}

func newCLGenesisData(genesisUnixTimestamp uint64, parentDirpath string, configYmlFilepath string, genesisSszFilepath string) *CLGenesisData {
	return &CLGenesisData{genesisUnixTimestamp: genesisUnixTimestamp, parentDirpath: parentDirpath, configYmlFilepath: configYmlFilepath, genesisSszFilepath: genesisSszFilepath}
}

func (paths *CLGenesisData) GetGenesisUnixTimestamp() uint64 {
	return paths.genesisUnixTimestamp
}
func (paths *CLGenesisData) GetParentDirpath() string {
	return paths.parentDirpath
}
func (paths *CLGenesisData) GetConfigYMLFilepath() string {
	return paths.configYmlFilepath
}
func (paths *CLGenesisData) GetGenesisSSZFilepath() string {
	return paths.genesisSszFilepath
}
