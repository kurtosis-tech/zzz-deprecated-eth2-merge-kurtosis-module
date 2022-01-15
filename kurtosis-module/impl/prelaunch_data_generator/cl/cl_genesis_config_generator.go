package cl

import (
	"github.com/kurtosis-tech/stacktrace"
	"io/ioutil"
	"path"
	"text/template"
)

const (

	// Indicates that Go should use the default tempdir
	outputTempdir = ""
	outputTempdirPattern = "cl-genesis-config"


	genesisGenerationConfigFilename = "config.yaml"
	genesisGenerationMnemonicsFilename = "mnemonics.yaml"
)


// Generate CL client genesis config files inside a tempdir on the module container
func generateClGenesisGenerationConfig(
	genesisGenerationConfigYmlTemplate *template.Template,
	genesisGenerationMnemonicsYmlTemplate *template.Template,
	genesisUnixTimestamp uint64,
	delay uint64,
	networkId string,
	totalTerminalDifficulty uint64,
	secondsPerSlot uint32,
	altairForkEpoch uint64,
	mergeForkEpoch uint64,
	preregisteredValidatorKeysMnemonic string,
	numValidatorKeysToPreregister uint32,
) (
	string,
	error,
) {
	templateData := &clGenesisConfigTemplateData{
		NetworkId:                          networkId,
		SecondsPerSlot:                     secondsPerSlot,
		UnixTimestamp:                      genesisUnixTimestamp,
		Delay:                              delay,
		TotalTerminalDifficulty:            totalTerminalDifficulty,
		AltairForkEpoch:                    altairForkEpoch,
		MergeForkEpoch:                     mergeForkEpoch,
		NumValidatorKeysToPreregister:      numValidatorKeysToPreregister,
		PreregisteredValidatorKeysMnemonic: preregisteredValidatorKeysMnemonic,
	}

	tempDirpath, err := ioutil.TempDir(outputTempdir, outputTempdirPattern)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred creating a temporary directory to hold the CL genesis config")
	}


	genesisGenerationConfigFilepath := path.Join(tempDirpath, genesisGenerationConfigFilename)
	if err := fillTemplate(genesisGenerationConfigYmlTemplate, templateData, genesisGenerationConfigFilepath); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred filling the CL genesis generation config YML template to '%v'", genesisGenerationConfigFilepath)
	}

	genesisGenerationMnemonicsFilepath := path.Join(tempDirpath, genesisGenerationMnemonicsFilename)
	if err := fillTemplate(genesisGenerationMnemonicsYmlTemplate, templateData, genesisGenerationMnemonicsFilepath); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred filling the CL genesis generation mnemonics YML template to '%v'", genesisGenerationMnemonicsFilepath)
	}

	return tempDirpath, nil
}

