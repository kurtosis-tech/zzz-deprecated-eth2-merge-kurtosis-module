package participant_network

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lighthouse"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lodestar"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/nimbus"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/prysm"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/teku"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/besu"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/erigon"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/geth"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/nethermind"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/mev_boost"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/cl_genesis"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
)

const (
	clClientServiceIdPrefix = "cl-client-"
	elClientServiceIdPrefix = "el-client-"
	mevBoostServiceIdPrefix = "mev-boost-"

	bootParticipantIndex = 0

	// The more nodes, the longer DAG generation takes so the longer we have to wait for a node to become available
	// TODO MAKE THIS CONFIGURABLE BASED ON ESTIMATED TIME-TO-DAG-GENERATION
	elClientMineWaiterMaxNumRetriesPerNode = uint32(120)
	elClientMineWaiterTimeBetweenRetries   = 5 * time.Second

	// The time that the CL genesis generation step takes to complete, based off what we've seen
	clGenesisDataGenerationTime = 2 * time.Minute

	// Each CL node takes about this time to start up and start processing blocks, so when we create the CL
	//  genesis data we need to set the genesis timestamp in the future so that nodes don't miss important slots
	// (e.g. Altair fork)
	// TODO Make this client-specific (currently this is Nimbus)
	clNodeStartupTime = 45 * time.Second
)

// To get clients to start as bootnodes, we pass in these values when starting them
var clClientContextForBootClClients *cl.CLClientContext = nil

func LaunchParticipantNetwork(
	ctx context.Context,
	// TODO this is a hack to allow for starting just the EL nodes! This will be fixed by Kurtosis product work
	shouldStartJustELNodes bool,
	enclaveCtx *enclaves.EnclaveContext,
	networkParams *module_io.NetworkParams,
	allParticipantSpecs []*module_io.ParticipantParams,
	globalLogLevel module_io.GlobalClientLogLevel,
	shouldWaitForMining bool,
) (
	resultParticipants []*Participant,
	resultClGenesisUnixTimestamp uint64,
	resultErr error,
) {
	numParticipants := uint32(len(allParticipantSpecs))

	// Parse all the templates we'll need first, so if an error is thrown it'll be thrown early
	elGenesisGenerationConfigTemplate, err := ioutil.ReadFile(static_files.ELGenesisGenerationConfigTemplateFilepath)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred reading the EL genesis generation config YAML template")
	}
	clGenesisConfigTemplate, err := ioutil.ReadFile(static_files.CLGenesisGenerationConfigTemplateFilepath)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred reading the CL genesis generation config YAML template")
	}
	clGenesisMnemonicsYmlTemplate, err := ioutil.ReadFile(static_files.CLGenesisGenerationMnemonicsTemplateFilepath)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred reading the CL mnemonics YAML template")
	}

	elGenesisGenerationConfigTemplateString := string(elGenesisGenerationConfigTemplate)
	clGenesisConfigTemplateString := string(clGenesisConfigTemplate)
	clGenesisMnemonicsYmlTemplateString := string(clGenesisMnemonicsYmlTemplate)

	// CL validator key generation is CPU-intensive, so we want to do the generation before any EL clients start mining
	//  (even though we only start the CL clients after the EL network is fully up & mining)
	logrus.Info("Generating validator keys....")
	clValidatorData, err := cl_validator_keystores.GenerateCLValidatorKeystores(
		ctx,
		enclaveCtx,
		networkParams.PreregisteredValidatorKeysMnemonic,
		numParticipants,
		networkParams.NumValidatorKeysPerNode,
	)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred generating CL validator keys")
	}
	logrus.Info("Successfully generated validator keys")

	// Per Pari's recommendation, we want to start all EL clients before any CL clients and wait until they're all mining blocks before
	//  we start the CL clients. This matches the real world, where Eth1 definitely exists before Eth2
	logrus.Info("Generating EL client genesis data...")
	elGenesisTimestamp := uint64(time.Now().Unix())
	elGenesisData, err := el_genesis.GenerateELGenesisData(
		ctx,
		enclaveCtx,
		elGenesisGenerationConfigTemplateString,
		elGenesisTimestamp,
		networkParams.NetworkID,
		networkParams.DepositContractAddress,
		networkParams.TotalTerminalDifficulty,
	)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred generating EL client genesis data")
	}
	logrus.Info("Successfully generated EL client genesis data")

	logrus.Info("Uploading Geth prefunded keys...")
	gethPrefundedKeysArtifactId, err := enclaveCtx.UploadFiles(static_files.GethPrefundedKeysDirpath)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred uploading the Geth prefunded keys to the enclave")
	}
	logrus.Info("Successfully uploaded Geth prefunded keys")

	logrus.Infof("Adding %v EL clients...", numParticipants)
	elClientLaunchers := map[module_io.ParticipantELClientType]el.ELClientLauncher{
		module_io.ParticipantELClientType_Geth: geth.NewGethELClientLauncher(
			elGenesisData,
			gethPrefundedKeysArtifactId,
			genesis_consts.PrefundedAccounts,
			networkParams.NetworkID,
		),
		module_io.ParticipantELClientType_Erigon: erigon.NewErigonELClientLauncher(
			elGenesisData,
			networkParams.NetworkID,
		),
		module_io.ParticipantELClientType_Nethermind: nethermind.NewNethermindELClientLauncher(
			elGenesisData,
			networkParams.TotalTerminalDifficulty,
		),
		module_io.ParticipantELClientType_Besu: besu.NewBesuELClientLauncher(
			elGenesisData,
			networkParams.NetworkID,
		),
	}
	allElClientContexts := []*el.ELClientContext{}
	for idx, participantSpec := range allParticipantSpecs {
		elClientType := participantSpec.ELClientType
		elLauncher, found := elClientLaunchers[elClientType]
		if !found {
			return nil, 0, stacktrace.NewError("No EL client launcher defined for EL client type '%v'", elClientType)
		}

		elClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", elClientServiceIdPrefix, idx))

		// Add EL client
		var newElClientCtx *el.ELClientContext
		newElClientCtx, err = elLauncher.Launch(
			enclaveCtx,
			elClientServiceId,
			participantSpec.ELClientImage,
			participantSpec.ELClientLogLevel,
			globalLogLevel,
			allElClientContexts,
			participantSpec.ELExtraParams,
		)
		if err != nil {
			return nil, 0, stacktrace.Propagate(err, "An error occurred launching EL client for participant %v", idx)
		}
		allElClientContexts = append(allElClientContexts, newElClientCtx)
		logrus.Infof("Added EL client %v of type '%v'", idx, elClientType)
	}
	logrus.Infof("Successfully added %v EL clients", numParticipants)
	// TODO This is a temporary hack to enable starting an EL-node-only network!
	//  Will be fixed by Kurtosis product work to make the module easily decomposable
	if shouldStartJustELNodes {
		resultParticipants := []*Participant{}
		for idx, participantSpec := range allParticipantSpecs {
			elClientCtx := allElClientContexts[idx]
			participant := NewParticipant(
				participantSpec.ELClientType,
				participantSpec.CLClientType,
				elClientCtx,
				nil,
				nil,
			)
			resultParticipants = append(resultParticipants, participant)
		}
		return resultParticipants, 0, nil
	}

	// We create the CL genesis data after the EL network is ready so that the CL genesis timestamp will be close
	//  to the time the CL nodes are started
	logrus.Info("Generating CL client genesis data...")
	// Set the genesis timestamp in the future so we don't start running slots until all the CL nodes are up
	clGenesisTimestamp := uint64(time.Now().Unix()) +
		uint64(clGenesisDataGenerationTime.Seconds()) +
		uint64(numParticipants)*uint64(clNodeStartupTime.Seconds())
	clGenesisData, err := cl_genesis.GenerateCLGenesisData(
		ctx,
		enclaveCtx,
		clGenesisConfigTemplateString,
		clGenesisMnemonicsYmlTemplateString,
		elGenesisData,
		clGenesisTimestamp,
		networkParams.NetworkID,
		networkParams.DepositContractAddress,
		networkParams.TotalTerminalDifficulty,
		networkParams.SecondsPerSlot,
		networkParams.AltairForkEpoch,
		networkParams.MergeForkEpoch,
		networkParams.PreregisteredValidatorKeysMnemonic,
		networkParams.NumValidatorKeysPerNode,
	)
	if err != nil {
		return nil, 0, stacktrace.Propagate(err, "An error occurred generating the CL client genesis data")
	}
	logrus.Info("Successfully generated CL client genesis data")

	logrus.Infof("Adding %v CL clients...", numParticipants)
	clClientLaunchers := map[module_io.ParticipantCLClientType]cl.CLClientLauncher{
		module_io.ParticipantCLClientType_Teku: teku.NewTekuCLClientLauncher(
			clGenesisData,
		),
		module_io.ParticipantCLClientType_Nimbus: nimbus.NewNimbusLauncher(
			clGenesisData,
		),
		module_io.ParticipantCLClientType_Lodestar: lodestar.NewLodestarClientLauncher(
			clGenesisData,
		),
		module_io.ParticipantCLClientType_Lighthouse: lighthouse.NewLighthouseCLClientLauncher(
			clGenesisData,
		),
		module_io.ParticipantCLClientType_Prysm: prysm.NewPrysmCLClientLauncher(
			clGenesisData,
			clValidatorData.PrysmPasswordArtifactUUid,
			clValidatorData.PrysmPasswordRelativeFilepath,
		),
	}
	preregisteredValidatorKeysForNodes := clValidatorData.PerNodeKeystores
	allMevBoostContexts := []*mev_boost.MEVBoostContext{}
	allClClientContexts := []*cl.CLClientContext{}
	for idx, participantSpec := range allParticipantSpecs {
		clClientType := participantSpec.CLClientType

		clLauncher, found := clClientLaunchers[clClientType]
		if !found {
			return nil, 0, stacktrace.NewError("No CL client launcher defined for CL client type '%v'", clClientType)
		}

		clClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", clClientServiceIdPrefix, idx))
		newClNodeValidatorKeystores := preregisteredValidatorKeysForNodes[idx]

		// Each CL node will be paired with exactly one EL node
		elClientCtx := allElClientContexts[idx]

		var mevBoostCtx *mev_boost.MEVBoostContext
		if participantSpec.BuilderNetworkParams != nil {
			mevBoostLauncher := mev_boost.MEVBoostLauncher{
				ShouldCheckRelay: true,
				RelayEndpoints:   participantSpec.BuilderNetworkParams.RelayEndpoints,
			}
			mevBoostServiceId := services.ServiceID(fmt.Sprintf("%v%v", mevBoostServiceIdPrefix, idx))
			mevBoostCtx, err = mevBoostLauncher.Launch(enclaveCtx, mevBoostServiceId, networkParams.NetworkID)
			if err != nil {
				return nil, 0, stacktrace.Propagate(
					err, fmt.Sprintf("could not start mev-boost sidecar with service ID '%v'", mevBoostServiceId),
				)
			}
		}
		allMevBoostContexts = append(allMevBoostContexts, mevBoostCtx)

		// Launch CL client
		var newClClientCtx *cl.CLClientContext
		var clClientLaunchErr error
		if idx == bootParticipantIndex {
			newClClientCtx, clClientLaunchErr = clLauncher.Launch(
				enclaveCtx,
				clClientServiceId,
				participantSpec.CLClientImage,
				participantSpec.CLClientLogLevel,
				globalLogLevel,
				clClientContextForBootClClients,
				elClientCtx,
				mevBoostCtx,
				newClNodeValidatorKeystores,
				participantSpec.BeaconExtraParams,
				participantSpec.ValidatorExtraParams,
			)
		} else {
			bootClClientCtx := allClClientContexts[bootParticipantIndex]
			newClClientCtx, clClientLaunchErr = clLauncher.Launch(
				enclaveCtx,
				clClientServiceId,
				participantSpec.CLClientImage,
				participantSpec.CLClientLogLevel,
				globalLogLevel,
				bootClClientCtx,
				elClientCtx,
				mevBoostCtx,
				newClNodeValidatorKeystores,
				participantSpec.BeaconExtraParams,
				participantSpec.ValidatorExtraParams,
			)
		}
		if clClientLaunchErr != nil {
			return nil, 0, stacktrace.Propagate(clClientLaunchErr, "An error occurred launching CL client for participant %v", idx)
		}

		allClClientContexts = append(allClClientContexts, newClClientCtx)
		logrus.Infof("Added CL client %v of type '%v'", idx, clClientType)
	}
	logrus.Infof("Successfully added %v CL clients", numParticipants)

	allParticipants := []*Participant{}
	for idx, participantSpec := range allParticipantSpecs {
		elClientType := participantSpec.ELClientType
		clClientType := participantSpec.CLClientType

		elClientCtx := allElClientContexts[idx]
		clClientCtx := allClClientContexts[idx]
		mevBoostCtx := allMevBoostContexts[idx]

		participant := NewParticipant(
			elClientType,
			clClientType,
			elClientCtx,
			clClientCtx,
			mevBoostCtx,
		)
		allParticipants = append(allParticipants, participant)
	}

	return allParticipants, clGenesisTimestamp, nil
}
