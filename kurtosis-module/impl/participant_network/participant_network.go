package participant_network
import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/log_levels"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl"
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
	ELClientType ParticipantELClientType
	CLClientType ParticipantCLClientType
}

func LaunchParticipantNetwork(
	enclaveCtx *enclaves.EnclaveContext,
	networkId string,
	elClientLaunchers map[ParticipantELClientType]el.ELClientLauncher,
	clClientLaunchers map[ParticipantCLClientType]cl.CLClientLauncher,
	allParticipantSpecs []*ParticipantSpec,
	preregisteredValidatorKeysForNodes []*cl2.NodeTypeKeystoreDirpaths,
	logLevel log_levels.ParticipantLogLevel,
) (
	[]*Participant,
	error,
) {
	numParticipants := len(allParticipantSpecs)

	// Per Pari's recommendation, we want to start all EL clients first and wait until they're all mining blocks before
	//  we start the CL clients. This matches the real world, where Eth1 definitely exists before Eth2
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

	logrus.Infof("Adding %v CL clients...", numParticipants)
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


/*
// Represents a network of virtual "participants", where each participant runs:
//  1) an EL client
//  2) a Beacon client
//  3) a validator client
type ParticipantNetwork struct {
	enclaveCtx *enclaves.EnclaveContext

	networkId string

	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths

	participants []*Participant

	elClientLaunchers map[ParticipantELClientType]el.ELClientLauncher
	clClientLaunchers map[ParticipantCLClientType]cl.CLClientLauncher

	mutex *sync.Mutex
}

func NewParticipantNetwork(
	enclaveCtx *enclaves.EnclaveContext,
	networkId string,
	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths,
	elClientLaunchers map[ParticipantELClientType]el.ELClientLauncher,
	clClientLaunchers map[ParticipantCLClientType]cl.CLClientLauncher,
) *ParticipantNetwork {
	return &ParticipantNetwork{
		enclaveCtx: enclaveCtx,
		networkId: networkId,
		preregisteredValidatorKeysForNodes: preregisteredValidatorKeysForNodes,
		participants: []*Participant{},
		elClientLaunchers: elClientLaunchers,
		clClientLaunchers: clClientLaunchers,
		mutex: &sync.Mutex{},
	}
}

func (network *ParticipantNetwork) AddParticipant(
	elClientType ParticipantELClientType,
	clClientType ParticipantCLClientType,
	logLevel log_levels.ParticipantLogLevel,
) (*Participant, error) {
	network.mutex.Lock()
	defer network.mutex.Unlock()

	elLauncher, found := network.elClientLaunchers[elClientType]
	if !found {
		return nil, stacktrace.NewError("No EL client launcher defined for EL client type '%v'", elClientType)
	}
	clLauncher, found := network.clClientLaunchers[clClientType]
	if !found {
		return nil, stacktrace.NewError("No CL client launcher defined for CL client type '%v'", clClientType)
	}

	newParticipantIdx := len(network.participants)
	elClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", elClientServiceIdPrefix, newParticipantIdx))
	clClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", clClientServiceIdPrefix, newParticipantIdx))
	newClNodeValidatorKeystores := network.preregisteredValidatorKeysForNodes[newParticipantIdx]

	// Add EL client
	var newElClientCtx *el.ELClientContext
	var elClientLaunchErr error
	if newParticipantIdx == bootParticipantIndex {
		newElClientCtx, elClientLaunchErr = elLauncher.Launch(
			network.enclaveCtx,
			elClientServiceId,
			logLevel,
			network.networkId,
			elClientContextForBootElClients,
		)
	} else {
		bootParticipant := network.participants[bootParticipantIndex]
		bootElClientCtx := bootParticipant.GetELClientContext()
		newElClientCtx, elClientLaunchErr = elLauncher.Launch(
			network.enclaveCtx,
			elClientServiceId,
			logLevel,
			network.networkId,
			bootElClientCtx,
		)
	}
	if elClientLaunchErr != nil {
		return nil, stacktrace.Propagate(elClientLaunchErr, "An error occurred launching EL client for participant %v", newParticipantIdx)
	}

	// Launch CL client
	var newClClientCtx *cl.CLClientContext
	var clClientLaunchErr error
	if newParticipantIdx == bootParticipantIndex {
		newClClientCtx, clClientLaunchErr = clLauncher.Launch(
			network.enclaveCtx,
			clClientServiceId,
			logLevel,
			clClientContextForBootClClients,
			newElClientCtx,
			newClNodeValidatorKeystores,
		)
	} else {
		bootParticipant := network.participants[bootParticipantIndex]
		bootClClientCtx := bootParticipant.GetCLClientContext()
		newClClientCtx, clClientLaunchErr = clLauncher.Launch(
			network.enclaveCtx,
			clClientServiceId,
			logLevel,
			bootClClientCtx,
			newElClientCtx,
			newClNodeValidatorKeystores,
		)
	}
	if clClientLaunchErr != nil {
		return nil, stacktrace.Propagate(clClientLaunchErr, "An error occurred launching CL client for participant %v", newParticipantIdx)
	}

	participant := NewParticipant(
		elClientType,
		clClientType,
		newElClientCtx,
		newClClientCtx,
	)
	network.participants = append(network.participants, participant)

	return participant, nil
}

 */