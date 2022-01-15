package cl

type CLPrelaunchData struct {
	genesisPaths              *CLGenesisPaths
	keystoreGenerationResults *GenerateKeystoresResult
}

func newCLPrelaunchData(genesisPaths *CLGenesisPaths, keystoreGenerationResults *GenerateKeystoresResult) *CLPrelaunchData {
	return &CLPrelaunchData{genesisPaths: genesisPaths, keystoreGenerationResults: keystoreGenerationResults}
}

func (data *CLPrelaunchData) GetCLGenesisPaths() *CLGenesisPaths {
	return data.genesisPaths
}
func (data *CLPrelaunchData) GetCLKeystoreGenerationResults() *GenerateKeystoresResult {
	return data.keystoreGenerationResults
}
