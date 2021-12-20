package prelaunch_data_generator

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"strings"
)

const (
	// Prysm keystores are encrypted with a password
	prysmPassword = "password"

	keystoresGenerationToolName = "eth2-val-tools"

	expectedExitCode = 0
)

// Generates keystores for the given number of nodes from the given mnemonic, where each keystore contains approximately
//  num_keys / num_nodes keys
func generateKeystores(
	serviceCtx *services.ServiceContext,
	mnemonic string,
	numPreregisteredValidators uint32,	// The number of validators that were preregistered during the creation of the CL genesis
	numNodes uint32,
) (
	*GenerateKeystoresResult,
	error,
){
	sharedDir := serviceCtx.GetSharedDirectory()
	startIndices, stopIndices, err := generateKeyStartAndStopIndices(numPreregisteredValidators, numNodes)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the validator key start & stop indices for the nodes")
	}

	allNodeKeystoreDirpaths := []*NodeTypeKeystoreDirpaths{}
	allSubcommandStrs := []string{}
	for i := uint32(0); i < numNodes; i++ {
		startIndex := startIndices[i]
		stopIndex := stopIndices[i]

		nodeKeystoresDirname := fmt.Sprintf("node-%v-keystores", i)
		nodeOutputSharedPath := sharedDir.GetChildPath(nodeKeystoresDirname)

		subcommandStr := fmt.Sprintf(
			"%v keystores --prysm-pass %v --out-loc %v --source-mnemonic \"%v\" --source-min %v --source-max %v",
			keystoresGenerationToolName,
			prysmPassword,
			nodeOutputSharedPath.GetAbsPathOnServiceContainer(),
			mnemonic,
			startIndex,
			stopIndex,
		)
		allSubcommandStrs = append(allSubcommandStrs, subcommandStr)

		nodeKeystoreDirpaths := NewNodeTypeKeystoreDirpathsFromOutputSharedPath(nodeOutputSharedPath)
		allNodeKeystoreDirpaths = append(allNodeKeystoreDirpaths, nodeKeystoreDirpaths)
	}

	commandStr := strings.Join(allSubcommandStrs, " && ")

	exitCode, output, err := serviceCtx.ExecCommand([]string{"sh", "-c", commandStr})
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred executing the following command to generate keystores for each node: %v", commandStr)
	}
	if exitCode != expectedExitCode {
		return nil, stacktrace.NewError(
			"Command '%v' to generate keystores for each node returned non-%v exit code %v and logs:\n%v",
			commandStr,
			expectedExitCode,
			exitCode,
			output,
		)
	}

	result := NewGenerateKeystoresResult(
		prysmPassword,
		allNodeKeystoreDirpaths,
	)

	return result, nil
}

func generateKeyStartAndStopIndices(
	numPreregisteredValidators uint32,
	numNodes uint32,
) (
	resultInclusiveKeyStartIndices []uint32,
	resultExclusiveKeyStopIndices []uint32,
	resultErr error,
){
	if (numNodes > numPreregisteredValidators) {
		return nil, nil, stacktrace.NewError(
			"Number of preregistered validators '%v' must be >= number of CL nodes '%v'",
			numPreregisteredValidators,
			numNodes,
		)
	}
	validatorsPerNode := numPreregisteredValidators / numNodes

	// If mod(num_validators / num_nodes) != 0, we have to give one of the nodes extra keys
	leftover := numPreregisteredValidators % numNodes

	inclusiveKeyStartIndices := []uint32{}
	exclusiveKeyStopIndices := []uint32{}

	lastEndIndex := uint32(0)
	for i := uint32(0); i < numNodes; i++ {
		rangeSize := validatorsPerNode
		if i == 0 {
			rangeSize = rangeSize + leftover
		}
		rangeStart := lastEndIndex
		rangeEnd := lastEndIndex + rangeSize
		inclusiveKeyStartIndices = append(inclusiveKeyStartIndices, rangeStart)
		exclusiveKeyStopIndices = append(exclusiveKeyStopIndices, rangeEnd)
		lastEndIndex = rangeEnd
	}
	return inclusiveKeyStartIndices, exclusiveKeyStopIndices, nil
}
