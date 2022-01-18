package cl_validator_keystores

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"strings"
	"time"
)

const (
	outputDirnamePrefix = "cl-validator-keystores-"

	// Prysm keystores are encrypted with a password
	prysmPassword = "password"

	keystoresGenerationToolName = "eth2-val-tools"

	expectedExitCode = 0
)

// Generates keystores for the given number of nodes from the given mnemonic, where each keystore contains approximately
//  num_keys / num_nodes keys
func GenerateCLValidatorKeystores(
	serviceCtx *services.ServiceContext,
	mnemonic string,
	numNodes uint32,
	numValidatorsPerNode uint32,
) (
	*GenerateKeystoresResult,
	error,
){
	sharedDir := serviceCtx.GetSharedDirectory()
	outputSharedDir := sharedDir.GetChildPath(fmt.Sprintf(
		"%v%v",
		outputDirnamePrefix,
		time.Now().Unix(),
	))

	allNodeKeystoreDirpaths := []*NodeTypeKeystoreDirpaths{}
	allSubcommandStrs := []string{}

	startIndex := uint32(0)
	stopIndex := numValidatorsPerNode
	for i := uint32(0); i < numNodes; i++ {
		nodeKeystoresDirname := fmt.Sprintf("node-%v-keystores", i)
		nodeOutputSharedPath := outputSharedDir.GetChildPath(nodeKeystoresDirname)
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

		startIndex = stopIndex
		stopIndex = stopIndex + numValidatorsPerNode
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