package impl

import (
	"encoding/json"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/forkmon"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/transaction_spammer"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"time"
)

const (
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


type Eth2KurtosisModule struct {
}

func NewEth2KurtosisModule() *Eth2KurtosisModule {
	return &Eth2KurtosisModule{}
}

func (e Eth2KurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	logrus.Info("Deserializing execute params...")
	paramsObj, err := module_io.DeserializeAndValidateParams(serializedParams)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred deserializing & validating the params")
	}
	networkParams := paramsObj.Network
	numParticipants := uint32(len(paramsObj.Participants))
	logrus.Info("Successfully deserialized execute params")

	logrus.Info("Creating prelaunch data generator...")
	prelaunchDataGeneratorCtx, err := prelaunch_data_generator.LaunchPrelaunchDataGenerator(
		enclaveCtx,
		networkParams.NetworkID,
		networkParams.DepositContractAddress,
		networkParams.TotalTerminalDifficulty,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the prelaunch data-generating container")
	}
	logrus.Info("Successfully created prelaunch data generator")

	logrus.Infof("Adding %v participants logging at level '%v'...", numParticipants, paramsObj.ClientLogLevel)
	participants, clGenesisUnixTimestamp, err := participant_network.LaunchParticipantNetwork(
		enclaveCtx,
		prelaunchDataGeneratorCtx,
		networkParams,
		paramsObj.Participants,
		paramsObj.ClientLogLevel,
	)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred launching a participant network of '%v' participants",
			numParticipants,
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
	forkmonConfigTemplate, err := static_files.ParseTemplate(static_files.ForkmonConfigTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred parsing forkmon config template file '%v'", static_files.ForkmonConfigTemplateFilepath)
	}
	forkmonPublicUrl, err := forkmon.LaunchForkmon(
		enclaveCtx,
		forkmonConfigTemplate,
		allClClientContexts,
		clGenesisUnixTimestamp,
		networkParams.SecondsPerSlot,
	)
	logrus.Info("Successfully launched forkmon")

	if paramsObj.WaitForFinalization {
		logrus.Info("Waiting for the first finalized epoch...")
		firstClClientCtx := allClClientContexts[0]
		firstClClientRestClient := firstClClientCtx.GetRESTClient()
		if err := waitUntilFirstFinalizedEpoch(firstClClientRestClient, networkParams.SecondsPerSlot, networkParams.SlotsPerEpoch); err != nil {
			return "", stacktrace.Propagate(err, "An error occurred waiting until the first finalized epoch occurred")
		}
		logrus.Info("First finalized epoch occurred successfully")
	}

	responseObj := &module_io.ExecuteResponse{
		ForkmonPublicURL: forkmonPublicUrl,
	}
	responseStr, err := json.MarshalIndent(responseObj, responseJsonLinePrefixStr, responseJsonLineIndentStr)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred serializing the following response object to JSON for returning: %+v", responseObj)
	}

	return string(responseStr), nil
}


func waitUntilFirstFinalizedEpoch(
	restClient *cl_client_rest_client.CLClientRESTClient,
	secondsPerSlot uint32,
	slotsPerEpoch uint32,
) error {
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
