package impl

import (
	"encoding/json"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/forkmon"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lighthouse"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lodestar"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/nimbus"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/prysm"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/teku"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/geth"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/nethermind"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/log_levels"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/transaction_spammer"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"path"
	"text/template"
	"time"
)

const (
	networkId = "3151908"

	// ----------------------------------- Params Constants -----------------------------------------
	defaultWaitForFinalization                                = false
	defaultClientLogLevel      log_levels.ParticipantLogLevel = log_levels.ParticipantLogLevel_Info
	// --------------------------------- End Params Constants ---------------------------------------

	// ----------------------------------- Prelaunch Data Constants -----------------------------------------
	// The number of validator keys that will be preregistered inside the CL genesis file when it's created
	numValidatorsToPreregister = 100

	// Seems to be hardcoded
	slotsPerEpoch = uint32(32)

	// If we drop this, things start to behave strangely, with slots that are of variable time lengths
	secondsPerSlot = uint32(12)

	// Altair must happen before merge fork must happen before terminal_total_difficulty is hit
	// See also: https://notes.ethereum.org/@ExXcnR0-SJGthjz1dwkA1A/H1MSKgm3F
	altairForkEpoch = uint64(1)  // Set per Parithosh's recommendation
	mergeForkEpoch = uint64(2)   // Set per Parithosh's recommendation

	// Once the total difficulty of all mined blocks crosses this threshold, the merge will be initiated
	// Must happen after the merge fork epoch on the Beacon chain
	totalTerminalDifficulty  = uint64(60000000)

	// This menmonic will a) be used to create keystores for all the types of validators that we have and b) be used to generate a CL genesis.ssz that has the children
	//  validator keys already preregistered as validators
	// See also:
	preregisteredValidatorKeysMnemonic = "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete"

	// TODO Clarify what units these are
	genesisDelay = 0

	depositContractAddress = "0x4242424242424242424242424242424242424242"
	// --------------------------------- End Genesis Config Constants ----------------------------------------

	// ----------------------------------- Static File Constants -----------------------------------------
	staticFilesDirpath                    = "/static-files"

	// Geth + CL genesis generation
	genesisGenerationConfigDirpath = staticFilesDirpath + "/genesis-generation-config"

	elGenesisGenerationConfigDirpath = genesisGenerationConfigDirpath + "/el"
	gethGenesisGenerationConfigYmlTemplateFilepath = elGenesisGenerationConfigDirpath + "/geth-genesis-config.yaml.tmpl"
	nethermindGenesisGenerationJsonTemplateFilepath = elGenesisGenerationConfigDirpath + "/nethermind-genesis.json.tmpl"

	clGenesisGenerationConfigDirpath = genesisGenerationConfigDirpath + "/cl"
	clGenesisGenerationConfigYmlTemplateFilepath = clGenesisGenerationConfigDirpath + "/config.yaml.tmpl"
	clGenesisGenerationMnemonicsYmlTemplateFilepath = clGenesisGenerationConfigDirpath + "/mnemonics.yaml.tmpl"

	// Prysm
	prysmPasswordTxtTemplateFilepath = staticFilesDirpath + "/prysm-password.txt.tmpl"

	// Forkmon config
	forkmonConfigTemplateFilepath = staticFilesDirpath + "/forkmon-config/config.toml.tmpl"
	// --------------------------------- End Static File Constants ----------------------------------------

	responseJsonLinePrefixStr = ""
	responseJsonLineIndentStr = "  "

	// TODO uncomment these when the module can either start a private network OR connect to an existing devnet
	// mergeDevnet3NetworkId = "1337602"
	// mergeDevnet3ClClientBootnodeEnr = "enr:-Iq4QKuNB_wHmWon7hv5HntHiSsyE1a6cUTK1aT7xDSU_hNTLW3R4mowUboCsqYoh1kN9v3ZoSu_WuvW9Aw0tQ0Dxv6GAXxQ7Nv5gmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk"

	// In normal operation, the finalized epoch will be this many epochs behind head
	expectedNumEpochsBehindHeadForFinalizedEpoch = uint64(3)
	firstHeadEpochWhereFinalizedEpochIsPossible = expectedNumEpochsBehindHeadForFinalizedEpoch + 1
	timeBetweenFinalizedEpochChecks = 5 * time.Second
	// TODO FIGURE OUT WHY THIS HAPPENS AND GET RID OF IT
	extraDelayBeforeSlotCountStartsIncreasing = 4 * time.Minute
)

var defaultParticipants = []*ParticipantParams{
	{
		ELClientType: participant_network.ParticipantELClientType_Geth,
		CLClientType: participant_network.ParticipantCLClientType_Nimbus,
	},
}

type ParticipantParams struct {
	ELClientType participant_network.ParticipantELClientType `json:"el"`
	CLClientType participant_network.ParticipantCLClientType `json:"cl"`
}
type ExecuteParams struct {
	// Participants
	Participants []*ParticipantParams	`json:"participants"`

	WaitForFinalization bool	`json:"waitForFinalization"`

	// The log level that the started clients should log at
	ClientLogLevel log_levels.ParticipantLogLevel `json:"logLevel"`
}

type ExecuteResponse struct {
	ForkmonPublicURL string	`json:"forkmonUrl"`
}

type Eth2KurtosisModule struct {
}

func NewEth2KurtosisModule() *Eth2KurtosisModule {
	return &Eth2KurtosisModule{}
}

func (e Eth2KurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	logrus.Info("Deserializing execute params...")
	paramsObj, err := deserializeAndValidateParams(serializedParams)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred deserializing & validating the params")
	}
	numParticipants := uint32(len(paramsObj.Participants))
	logrus.Info("Successfully deserialized execute params")

	logrus.Info("Generating prelaunch data...")
	genesisUnixTimestamp := time.Now().Unix()
	gethGenesisConfigTemplate, err := parseTemplate(gethGenesisGenerationConfigYmlTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the Geth genesis generation config YAML template")
	}
	clGenesisConfigTemplate, err := parseTemplate(clGenesisGenerationConfigYmlTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the CL genesis generation config YAML template")
	}
	clGenesisMnemonicsYmlTemplate, err := parseTemplate(clGenesisGenerationMnemonicsYmlTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the CL mnemonics YAML template")
	}
	nethermindGenesisJsonTemplate, err := parseTemplate(nethermindGenesisGenerationJsonTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the Nethermind genesis json template")
	}
	prysmPasswordTxtTemplate, err := parseTemplate(prysmPasswordTxtTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing the Prysm password txt template")
	}
	prelaunchData, err := prelaunch_data_generator.GeneratePrelaunchData(
		enclaveCtx,
		gethGenesisConfigTemplate,
		nethermindGenesisJsonTemplate,
		clGenesisConfigTemplate,
		clGenesisMnemonicsYmlTemplate,
		preregisteredValidatorKeysMnemonic,
		numValidatorsToPreregister,
		numParticipants,
		genesisUnixTimestamp,
		genesisDelay,
		networkId,
		secondsPerSlot,
		altairForkEpoch,
		mergeForkEpoch,
		totalTerminalDifficulty,
		preregisteredValidatorKeysMnemonic,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the Ethereum genesis generator Service")
	}
	logrus.Info("Successfully generated prelaunch data")

	logrus.Info("Creating EL & CL client launchers...")
	elClientLaunchers := map[participant_network.ParticipantELClientType]el.ELClientLauncher{
		participant_network.ParticipantELClientType_Geth: geth.NewGethELClientLauncher(
			prelaunchData.GethELGenesisJsonFilepathOnModuleContainer,
			genesis_consts.PrefundedAccounts,
		),
		participant_network.ParticipantELClientType_Nethermind: nethermind.NewNethermindELClientLauncher(
			prelaunchData.NethermindGenesisJsonFilepathOnModuleContainer,
			totalTerminalDifficulty,
		),
	}
	clGenesisPaths := prelaunchData.CLGenesisPaths
	clClientLaunchers := map[participant_network.ParticipantCLClientType]cl.CLClientLauncher{
		participant_network.ParticipantCLClientType_Teku: teku.NewTekuCLClientLauncher(
			clGenesisPaths.GetConfigYMLFilepath(),
			clGenesisPaths.GetGenesisSSZFilepath(),
			numParticipants,
		),
		participant_network.ParticipantCLClientType_Nimbus: nimbus.NewNimbusLauncher(
			clGenesisPaths.GetParentDirpath(),
		),
		participant_network.ParticipantCLClientType_Lodestar: lodestar.NewLodestarClientLauncher(
			clGenesisPaths.GetConfigYMLFilepath(),
			clGenesisPaths.GetGenesisSSZFilepath(),
			numParticipants,
		),
		participant_network.ParticipantCLClientType_Lighthouse: lighthouse.NewLighthouseCLClientLauncher(
			clGenesisPaths.GetParentDirpath(),
			numParticipants,
		 ),
		participant_network.ParticipantCLClientType_Prysm: prysm.NewPrysmCLCLientLauncher(
			clGenesisPaths.GetConfigYMLFilepath(),
			clGenesisPaths.GetGenesisSSZFilepath(),
			prelaunchData.KeystoresGenerationResult.PrysmPassword,
			prysmPasswordTxtTemplate,
			numParticipants,
		),
	}
	logrus.Info("Successfully created EL & CL client launchers")

	logrus.Infof("Adding %v participants logging at level '%v'...", numParticipants, paramsObj.ClientLogLevel)
	allParticipantSpecs := []*participant_network.ParticipantSpec{}
	for _, participantParams := range paramsObj.Participants {
		// Don't need to validate because we already did when deserializing
		elClientType := participantParams.ELClientType
		clClientType := participantParams.CLClientType

		participantSpec := &participant_network.ParticipantSpec{
			ELClientType: elClientType,
			CLClientType: clClientType,
		}
		allParticipantSpecs = append(allParticipantSpecs, participantSpec)
	}
	participants, err := participant_network.LaunchParticipantNetwork(
		enclaveCtx,
		networkId,
		elClientLaunchers,
		clClientLaunchers,
		allParticipantSpecs,
		prelaunchData.KeystoresGenerationResult.PerNodeKeystoreDirpaths,
		paramsObj.ClientLogLevel,
	)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred launching a participant network of '%v' participants",
			len(allParticipantSpecs),
		 )
	}
	allElClientContexts := []*el.ELClientContext{}
	allClClientContexts := []*cl.CLClientContext{}
	for _, participant := range participants {
		allElClientContexts = append(allElClientContexts, participant.GetELClientContext())
		allClClientContexts = append(allClClientContexts, participant.GetCLClientContext())
	}
	logrus.Infof("Successfully added %v participants", numParticipants)

	logrus.Info("Launching transaction spammer...")
	if err := transaction_spammer.LaunchTransanctionSpammer(
		enclaveCtx,
		genesis_consts.PrefundedAccounts,
		// TODO Upgrade the transaction spammer so it can take in multiple EL client addresses
		allElClientContexts[0],
	); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the transaction spammer")
	}
	logrus.Info("Successfully launched transaction spammer")

	logrus.Info("Launching forkmon...")
	forkmonConfigTemplate, err := parseTemplate(forkmonConfigTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing forkmon config template file '%v'", forkmonConfigTemplateFilepath)
	}
	forkmonPublicUrl, err := forkmon.LaunchForkmon(
		enclaveCtx,
		forkmonConfigTemplate,
		allClClientContexts,
		genesisUnixTimestamp,
		secondsPerSlot,
	)
	logrus.Info("Successfully launched forkmon")

	if paramsObj.WaitForFinalization {
		logrus.Info("Waiting for the first finalized epoch...")
		firstClClientCtx := allClClientContexts[0]
		firstClClientRestClient := firstClClientCtx.GetRESTClient()
		if err := waitUntilFirstFinalizedEpoch(firstClClientRestClient); err != nil {
			return "", stacktrace.Propagate(err, "An error occurred waiting until the first finalized epoch occurred")
		}
		logrus.Info("First finalized epoch occurred successfully")
	}

	responseObj := &ExecuteResponse{
		ForkmonPublicURL: forkmonPublicUrl,
	}
	responseStr, err := json.MarshalIndent(responseObj, responseJsonLinePrefixStr, responseJsonLineIndentStr)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred serializing the following response object to JSON for returning: %+v", responseObj)
	}

	return string(responseStr), nil
}

func deserializeAndValidateParams(paramsStr string) (*ExecuteParams, error) {
	paramsObj := &ExecuteParams{
		Participants:        defaultParticipants,
		WaitForFinalization: defaultWaitForFinalization,
		ClientLogLevel:      defaultClientLogLevel,
	}
	if err := json.Unmarshal([]byte(paramsStr), paramsObj); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred deserializing the serialized params")
	}
	if _, found := log_levels.ValidParticipantLogLevels[paramsObj.ClientLogLevel]; !found {
		return nil, stacktrace.NewError("Unrecognized client log level '%v'", paramsObj.ClientLogLevel)
	}
	if len(paramsObj.Participants) == 0 {
		return nil, stacktrace.NewError("At least one participant is required")
	}
	for idx, participant := range paramsObj.Participants {
		if idx == 0 && participant.ELClientType == participant_network.ParticipantELClientType_Nethermind {
			return nil, stacktrace.NewError("Cannot use a Nethermind client for the first participant because Nethermind clients don't mine on Eth1")
		}

		elClientType := participant.ELClientType
		if _, found := participant_network.ValidParticipantELClientTypes[elClientType]; !found {
			return nil, stacktrace.NewError("Participant %v declares unrecognized EL client type '%v'", idx, elClientType)
		}

		clClientType := participant.CLClientType
		if _, found := participant_network.ValidParticipantCLClientTypes[clClientType]; !found {
			return nil, stacktrace.NewError("Participant %v declares unrecognized CL client type '%v'", idx, clClientType)
		}
	}
	return paramsObj, nil
}

func parseTemplate(filepath string) (*template.Template, error) {
	tmpl, err := template.New(
		// For some reason, the template name has to match the basename of the file:
		//  https://stackoverflow.com/questions/49043292/error-template-is-an-incomplete-or-empty-template
		path.Base(filepath),
	).ParseFiles(
		filepath,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing template file '%v'", filepath)
	}
	return tmpl, nil
}

func waitUntilFirstFinalizedEpoch(restClient *cl_client_rest_client.CLClientRESTClient) error {
	// If we wait long enough that we might be in this epoch, we've waited too long - finality should already have happened
	waitedTooLongEpoch := firstHeadEpochWhereFinalizedEpochIsPossible + 1
	timeoutSeconds := waitedTooLongEpoch * uint64(slotsPerEpoch) * uint64(secondsPerSlot)
	timeout := time.Duration(timeoutSeconds) * time.Second + extraDelayBeforeSlotCountStartsIncreasing
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		currentSlot, err := restClient.GetCurrentSlot()
		if err != nil {
			return stacktrace.Propagate(err, "An error occurred getting the current slot using the REST client, which should never happen")
		}
		currentEpoch := currentSlot / uint64(slotsPerEpoch)
		finalizedEpoch, err := restClient.GetFinalizedEpoch()
		if err != nil {
			return stacktrace.Propagate(err, "An error occurred getting the finalized epoch using the REST client, which should never happen")
		}
		if finalizedEpoch > 0 && finalizedEpoch + expectedNumEpochsBehindHeadForFinalizedEpoch == currentEpoch {
			return nil
		}
		logrus.Debugf(
			"Finalized epoch hasn't occurred yet; current slot = '%v', current epoch = '%v', and finalized epoch = '%v'",
			currentSlot,
			currentEpoch,
			finalizedEpoch,
		 )
		time.Sleep(timeBetweenFinalizedEpochChecks)
	}
	return stacktrace.NewError("Waited for %v for the finalized epoch to be %v epochs behind the current epoch, but it didn't happen", timeout, expectedNumEpochsBehindHeadForFinalizedEpoch)
}
