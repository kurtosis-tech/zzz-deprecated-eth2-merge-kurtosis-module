package cl_genesis

type CLGenesisData struct {
	// genesis_timestamp + delay
	networkStartUnixTimestamp uint64

	// Path to the directory holding all the CL genesis files
	parentDirpath string

	configYmlFilepath string

	genesisSszFilepath string
}

func newCLGenesisData(networkStartUnixTimestamp uint64, parentDirpath string, configYmlFilepath string, genesisSszFilepath string) *CLGenesisData {
	return &CLGenesisData{networkStartUnixTimestamp: networkStartUnixTimestamp, parentDirpath: parentDirpath, configYmlFilepath: configYmlFilepath, genesisSszFilepath: genesisSszFilepath}
}

func (paths *CLGenesisData) GetNetworkStartUnixTimestamp() uint64 {
	return paths.networkStartUnixTimestamp
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
