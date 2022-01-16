package participant_network
import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lighthouse"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/lodestar"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/nimbus"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/prysm"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/teku"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/geth"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el/nethermind"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/genesis_consts"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	clClientServiceIdPrefix = "cl-client-"
	elClientServiceIdPrefix = "el-client-"

	bootParticipantIndex = 0

	// The more nodes, the longer DAG generation takes so the longer we have to wait for a node to become available
	// TODO MAKE THIS CONFIGURABLE BASED ON ESTIMATED TIME-TO-DAG-GENERATION
	elClientMineWaiterMaxNumRetriesPerNode = uint32(120)
	elClientMineWaiterTimeBetweenRetries = 5 * time.Second
)

// To get clients to start as bootnodes, we pass in these values when starting them
var elClientContextForBootElClients *el.ELClientContext = nil
var clClientContextForBootClClients *cl.CLClientContext = nil

type ParticipantSpec struct {
	ELClientType module_io.ParticipantELClientType
	CLClientType module_io.ParticipantCLClientType
}

func LaunchParticipantNetwork(
	enclaveCtx *enclaves.EnclaveContext,
	prelaunchDataGeneratorCtx *prelaunch_data_generator.PrelaunchDataGeneratorContext,
	networkParams *module_io.NetworkParams,
	allParticipantSpecs []*ParticipantSpec,
	logLevel module_io.ParticipantLogLevel,
) (
	[]*Participant,
	error,
) {
	numParticipants := uint32(len(allParticipantSpecs))

	// Parse all the templates we'll need first, so if an error is thrown it'll be thrown early
	chainspecAndGethGenesisGenerationConfigTemplate, err := static_files.ParseTemplate(static_files.ChainspecAndGethGenesisGenerationConfigTemplateFilepath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing the Geth genesis generation config YAML template")
	}
	clGenesisConfigTemplate, err := static_files.ParseTemplate(static_files.CLGenesisGenerationConfigTemplateFilepath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing the CL genesis generation config YAML template")
	}
	clGenesisMnemonicsYmlTemplate, err := static_files.ParseTemplate(static_files.CLGenesisGenerationMnemonicsTemplateFilepath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing the CL mnemonics YAML template")
	}
	nethermindGenesisJsonTemplate, err := static_files.ParseTemplate(static_files.NethermindGenesisGenerationJsonTemplateFilepath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing the Nethermind genesis json template")
	}

	// Per Pari's recommendation, we want to start all EL clients first and wait until they're all mining blocks before
	//  we start the CL clients. This matches the real world, where Eth1 definitely exists before Eth2
	logrus.Info("Creating EL client launchers...")
	elPrelaunchData, err := prelaunchDataGeneratorCtx.GenerateELData(
		chainspecAndGethGenesisGenerationConfigTemplate,
		nethermindGenesisJsonTemplate,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating EL client prelaunch data")
	}
	elClientLaunchers := map[module_io.ParticipantELClientType]el.ELClientLauncher{
		module_io.ParticipantELClientType_Geth: geth.NewGethELClientLauncher(
			elPrelaunchData.GetGethGenesisJsonFilepath(),
			genesis_consts.PrefundedAccounts,
		),
		module_io.ParticipantELClientType_Nethermind: nethermind.NewNethermindELClientLauncher(
			elPrelaunchData.GetNethermindGenesisJsonFilepath(),
			totalTerminalDifficulty,
		),
	}
	logrus.Info("Successfully created EL client launchers")

	logrus.Infof("Adding %v EL clients...", numParticipants)
	allElClientContexts := []*el.ELClientContext{}
	for idx, participantSpec := range allParticipantSpecs {
		elClientType := participantSpec.ELClientType
		elLauncher, found := elClientLaunchers[elClientType]
		if !found {
			return nil, stacktrace.NewError("No EL client launcher defined for EL client type '%v'", elClientType)
		}

		elClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", elClientServiceIdPrefix, idx))

		// Add EL client
		var newElClientCtx *el.ELClientContext
		var elClientLaunchErr error
		if idx == bootParticipantIndex {
			newElClientCtx, elClientLaunchErr = elLauncher.Launch(
				enclaveCtx,
				elClientServiceId,
				logLevel,
				networkId,
				elClientContextForBootElClients,
			)
		} else {
			bootElClientCtx := allElClientContexts[bootParticipantIndex]
			newElClientCtx, elClientLaunchErr = elLauncher.Launch(
				enclaveCtx,
				elClientServiceId,
				logLevel,
				networkId,
				bootElClientCtx,
			)
		}
		if elClientLaunchErr != nil {
			return nil, stacktrace.Propagate(elClientLaunchErr, "An error occurred launching EL client for participant %v", idx)
		}
		allElClientContexts = append(allElClientContexts, newElClientCtx)
		logrus.Infof("Added EL client %v of type '%v'", idx, elClientType)
	}
	logrus.Infof("Successfully added %v EL clients", numParticipants)

	// Wait for all EL clients to start mining before we proceed with adding the CL clients
	logrus.Infof("Waiting for all EL clients to start mining before adding CL clients...")
	perNodeNumRetries := uint32(numParticipants) * elClientMineWaiterMaxNumRetriesPerNode
	for idx, elClientCtx := range allElClientContexts {
		miningWaiter := elClientCtx.GetMiningWaiter()
		if err := miningWaiter.WaitForMining(
			perNodeNumRetries,
			elClientMineWaiterTimeBetweenRetries,
		 ); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"EL client %v didn't start mining even after %v retries with %v between retries",
				idx,
				perNodeNumRetries,
				elClientMineWaiterTimeBetweenRetries,
			 )
		}
		logrus.Infof("EL client %v has begun mining", idx)
	}
	logrus.Infof("All EL clients have started mining")

	// We create the CL genesis data after the EL network is ready so that the CL genesis timestamp will be close
	//  to the time the CL nodes are started
	logrus.Info("Creating CL client launchers...")
	clPrelaunchData, err := prelaunchDataGeneratorCtx.GenerateCLData(
		clGenesisConfigTemplate,
		clGenesisMnemonicsYmlTemplate,
		secondsPerSlot,
		altairForkEpoch,
		mergeForkEpoch,
		preregisteredValidatorKeysMnemonic,
		numValidatorsToPreregister,
		preregisteredValidatorKeysMnemonic,
		numParticipants,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the CL client prelaunch data")
	}
	clGenesisPaths := clPrelaunchData.GetCLGenesisPaths()
	clValidatorKeystoreGenerationResults := clPrelaunchData.GetCLKeystoreGenerationResults()
	clClientLaunchers := map[module_io.ParticipantCLClientType]cl.CLClientLauncher{
		module_io.ParticipantCLClientType_Teku: teku.NewTekuCLClientLauncher(
			clGenesisPaths.GetConfigYMLFilepath(),
			clGenesisPaths.GetGenesisSSZFilepath(),
			numParticipants,
		),
		module_io.ParticipantCLClientType_Nimbus: nimbus.NewNimbusLauncher(
			clGenesisPaths.GetParentDirpath(),
		),
		module_io.ParticipantCLClientType_Lodestar: lodestar.NewLodestarClientLauncher(
			clGenesisPaths.GetConfigYMLFilepath(),
			clGenesisPaths.GetGenesisSSZFilepath(),
			numParticipants,
		),
		module_io.ParticipantCLClientType_Lighthouse: lighthouse.NewLighthouseCLClientLauncher(
			clGenesisPaths.GetParentDirpath(),
			numParticipants,
		),
		module_io.ParticipantCLClientType_Prysm: prysm.NewPrysmCLCLientLauncher(
			clGenesisPaths.GetConfigYMLFilepath(),
			clGenesisPaths.GetGenesisSSZFilepath(),
			clValidatorKeystoreGenerationResults.PrysmPassword,
			numParticipants,
		),
	}
	logrus.Info("Successfully created CL client launchers")

	logrus.Infof("Adding %v CL clients...", numParticipants)
	preregisteredValidatorKeysForNodes := clValidatorKeystoreGenerationResults.PerNodeKeystoreDirpaths
	allClClientContexts := []*cl.CLClientContext{}
	for idx, participantSpec := range allParticipantSpecs {
		clClientType := participantSpec.CLClientType

		clLauncher, found := clClientLaunchers[clClientType]
		if !found {
			return nil, stacktrace.NewError("No CL client launcher defined for CL client type '%v'", clClientType)
		}

		clClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", clClientServiceIdPrefix, idx))
		newClNodeValidatorKeystores := preregisteredValidatorKeysForNodes[idx]

		// Each CL node will be paired with exactly one EL node
		elClientCtx := allElClientContexts[idx]

		// Launch CL client
		var newClClientCtx *cl.CLClientContext
		var clClientLaunchErr error
		if idx == bootParticipantIndex {
			newClClientCtx, clClientLaunchErr = clLauncher.Launch(
				enclaveCtx,
				clClientServiceId,
				logLevel,
				clClientContextForBootClClients,
				elClientCtx,
				newClNodeValidatorKeystores,
			)
		} else {
			bootClClientCtx := allClClientContexts[bootParticipantIndex]
			newClClientCtx, clClientLaunchErr = clLauncher.Launch(
				enclaveCtx,
				clClientServiceId,
				logLevel,
				bootClClientCtx,
				elClientCtx,
				newClNodeValidatorKeystores,
			)
		}
		if clClientLaunchErr != nil {
			return nil, stacktrace.Propagate(clClientLaunchErr, "An error occurred launching CL client for participant %v", idx)
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

		participant := NewParticipant(
			elClientType,
			clClientType,
			elClientCtx,
			clClientCtx,
		 )
		allParticipants = append(allParticipants, participant)
	}

	return allParticipants, nil
}