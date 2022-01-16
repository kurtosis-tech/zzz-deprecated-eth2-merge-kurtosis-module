package cl

type CLPrelaunchData struct {
	genesisUnixTimestamp uint64
	genesisPaths              *CLGenesisPaths
	keystoreGenerationResults *GenerateKeystoresResult
}

func newCLPrelaunchData(genesisUnixTimestamp uint64, genesisPaths *CLGenesisPaths, keystoreGenerationResults *GenerateKeystoresResult) *CLPrelaunchData {
	return &CLPrelaunchData{genesisUnixTimestamp: genesisUnixTimestamp, genesisPaths: genesisPaths, keystoreGenerationResults: keystoreGenerationResults}
}

func (data *CLPrelaunchData) GetGenesisUnixTimestamp() uint64 {
	return data.genesisUnixTimestamp
}
func (data *CLPrelaunchData) GetCLGenesisPaths() *CLGenesisPaths {
	return data.genesisPaths
}
func (data *CLPrelaunchData) GetCLKeystoreGenerationResults() *GenerateKeystoresResult {
	return data.keystoreGenerationResults
}
