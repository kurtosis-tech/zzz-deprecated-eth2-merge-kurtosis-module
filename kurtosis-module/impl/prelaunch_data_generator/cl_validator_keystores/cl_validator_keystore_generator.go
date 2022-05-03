package cl_validator_keystores

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/prelaunch_data_generator_launcher"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"path"
	"strings"
)

const (
	nodeKeystoresOutputDirpathFormatStr = "/node-%v-keystores"

	// Prysm keystores are encrypted with a password
	prysmPassword                    = "password"
	prysmPasswordFilepathOnGenerator = "/tmp/prysm-password.txt"

	keystoresGenerationToolName = "eth2-val-tools"

	successfulExecCmdExitCode = 0

	rawKeysDirname = "keys"
	rawSecretsDirname = "secrets"

	lodestarSecretsDirname = "lodestar-secrets"

	nimbusKeysDirname = "nimbus-keys"
	prysmDirname = "prysm"

	tekuKeysDirname = "teku-keys"
	tekuSecretsDirname = "teku-secrets"
)

// Generates keystores for the given number of nodes from the given mnemonic, where each keystore contains approximately
//  num_keys / num_nodes keys
func GenerateCLValidatorKeystores(
	ctx context.Context,
	enclaveCtx *enclaves.EnclaveContext,
	mnemonic string,
	numNodes uint32,
	numValidatorsPerNode uint32,
) (
	*GenerateKeystoresResult,
	error,
) {
	serviceCtx, err := prelaunch_data_generator_launcher.LaunchPrelaunchDataGenerator(
		enclaveCtx,
		map[services.FilesArtifactID]string{},
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the generator container")
	}
	defer func() {
		serviceId := serviceCtx.GetServiceID()
		if err := enclaveCtx.RemoveService(serviceId, 0); err != nil {
			logrus.Warnf("Tried to remove prelaunch data generator service '%v', but doing so threw an error:\n%v", serviceId, err)
		}
	}()

	allOutputDirpaths := []string{}
	allSubCommandStrs := []string{}

	// TODO Parallelize this to increase perf, which will require Docker exec operations not holding the Kurtosis mutex!
	startIndex := uint32(0)
	stopIndex := numValidatorsPerNode
	for i := uint32(0); i < numNodes; i++ {
		outputDirpath := fmt.Sprintf(
			nodeKeystoresOutputDirpathFormatStr,
			i,
		)
		generateKeystoresCmd := fmt.Sprintf(
			"%v keystores --insecure --prysm-pass %v --out-loc %v --source-mnemonic \"%v\" --source-min %v --source-max %v",
			keystoresGenerationToolName,
			prysmPassword,
			outputDirpath,
			mnemonic,
			startIndex,
			stopIndex,
		)
		allSubCommandStrs = append(allSubCommandStrs, generateKeystoresCmd)
		allOutputDirpaths = append(allOutputDirpaths, outputDirpath)

		startIndex = stopIndex
		stopIndex = stopIndex + numValidatorsPerNode
	}

	commandStr := strings.Join(allSubCommandStrs, " && ")

	if err := execCommand(serviceCtx, []string{"sh", "-c", commandStr}); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred executing keystore generation command: %+v", commandStr)
	}

	// Store outputs into files artifacts
	keystoreFiles := []*KeystoreFiles{}
	for idx, outputDirpath := range allOutputDirpaths {
		artifactId, err := enclaveCtx.StoreServiceFiles(ctx, serviceCtx.GetServiceID(), outputDirpath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred storing keystore files at '%v' for node #%v", outputDirpath, idx)
		}

		// This is necessary because the way Kurtosis currently implements artifact-storing is
		baseDirnameInArtifact := path.Base(outputDirpath)
		toAdd := NewKeystoreFiles(
			artifactId,
			path.Join(baseDirnameInArtifact, rawKeysDirname),
			path.Join(baseDirnameInArtifact, rawSecretsDirname),
			path.Join(baseDirnameInArtifact, lodestarSecretsDirname),
			path.Join(baseDirnameInArtifact, nimbusKeysDirname),
			path.Join(baseDirnameInArtifact, prysmDirname),
			path.Join(baseDirnameInArtifact, tekuKeysDirname),
			path.Join(baseDirnameInArtifact, tekuSecretsDirname),
		)

		keystoreFiles = append(keystoreFiles, toAdd)
	}

	writePrysmPasswordFileCmd := []string{
		"sh",
		"-c",
		fmt.Sprintf(
			"echo '%v' > %v",
			prysmPassword,
			prysmPasswordFilepathOnGenerator,
		),
	}
	if err := execCommand(serviceCtx, writePrysmPasswordFileCmd); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred writing the Prysm password to file '%v' on the generator", prysmPasswordFilepathOnGenerator)
	}
	prysmPasswordArtifactId, err := enclaveCtx.StoreServiceFiles(ctx, serviceCtx.GetServiceID(), prysmPasswordFilepathOnGenerator)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred storing the Prysm password file at '%v'", prysmPasswordFilepathOnGenerator)
	}

	result := NewGenerateKeystoresResult(
		prysmPasswordArtifactId,
		path.Base(prysmPasswordFilepathOnGenerator),
		keystoreFiles,
	)

	return result, nil
}

func execCommand(serviceCtx *services.ServiceContext, cmd []string) error {
	exitCode, output, err := serviceCtx.ExecCommand(cmd)
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred executing command '%+v' on the generator container",
			cmd,
		)
	}
	if exitCode != successfulExecCmdExitCode {
		return stacktrace.NewError(
			"Command '%+v' should have returned %v but returned %v with the following output:\n%v",
			cmd,
			successfulExecCmdExitCode,
			exitCode,
			output,
		)
	}
	return nil
}
