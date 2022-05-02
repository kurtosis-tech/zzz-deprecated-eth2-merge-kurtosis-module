package prelaunch_data_generator

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-engine-api-lib/api/golang/lib/kurtosis_context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

const (
	runKurtosisTestsEnvVar	= "RUN_KURTOSIS_TESTS"

	enclaveIdPrefix = "test-prelaunch-genesis-generation-"
	isPartitioningEnabled = false

	// Relative to the directory this file is inside
	staticFilesRelDirpath = "../../static_files"

	// Relative to the static files directory
	genesisGenerationConfigRelDirpath = "genesis-generation-config"
	elGenerationConfigRelDirpath = genesisGenerationConfigRelDirpath + "/el"
	gethGenesisConfigRelFilepath = elGenerationConfigRelDirpath + "/genesis-config.yaml.tmpl"
	clGenerationConfigRelDirpath = genesisGenerationConfigRelDirpath + "/cl"
	clGenesisConfigRelFilepath = clGenerationConfigRelDirpath + "/config.yaml.tmpl"
	clGenesisMnemonicsRelFilepath = clGenerationConfigRelDirpath + "/mnemonics.yaml.tmpl"
)

func TestPrelaunchGenesisGeneration(t *testing.T) {
	// TODO COMMENT
	/*
	if len(os.Getenv(runKurtosisTestsEnvVar)) == 0 {
		t.SkipNow()
	}

	 */

	// Go test always runs in the directory that this file is in
	pwd, err := os.Getwd()
	require.NoError(t, err)
	gethGenesisConfigTemplate, err := static_files.ParseTemplate(path.Join(
		pwd,
		staticFilesRelDirpath,
		gethGenesisConfigRelFilepath,
	))
	require.NoError(t, err)
	genesisConfigTemplate, err := static_files.ParseTemplate(path.Join(
		pwd,
		staticFilesRelDirpath,
		clGenesisConfigRelFilepath,
	))
	require.NoError(t, err)
	genesisMnemonicsTemplate, err := static_files.ParseTemplate(path.Join(
		pwd,
		staticFilesRelDirpath,
		clGenesisMnemonicsRelFilepath,
	))
	require.NoError(t, err)

	// Create enclave
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	require.NoError(t, err)
	enclaveId := enclaves.EnclaveID(fmt.Sprintf(
		"%v%v",
		enclaveIdPrefix,
		time.Now().Unix(),
	))
	enclaveCtx, err := kurtosisCtx.CreateEnclave(context.Background(), enclaveId, isPartitioningEnabled)
	require.NoError(t, err)
	defer func() {
		if err := kurtosisCtx.StopEnclave(context.Background(), enclaveId); err != nil {
			logrus.Errorf("We tried to stop the enclave we created, '%v', but an error occurred:\n%v", enclaveId, err)
			logrus.Errorf("ACTION REQUIRED: You'll need to stop enclave '%v' manually!", enclaveId)
		}
	}()

	executeParams := module_io.GetDefaultExecuteParams()
	networkParams := executeParams.Network
	participantParams := executeParams.Participants

	dataGeneratorCtx, err := LaunchPrelaunchDataGenerator(
		enclaveCtx,
		networkParams.NetworkID,
		networkParams.DepositContractAddress,
		networkParams.TotalTerminalDifficulty,
		networkParams.PreregisteredValidatorKeysMnemonic,
	)
	require.NoError(t, err)

	elGenesisData, err := el_genesis.GenerateELGenesisData(
		context.Background(),
		enclaveCtx,
		gethGenesisConfigTemplate,
		uint64(time.Now().Unix()),
		networkParams.NetworkID,
		networkParams.DepositContractAddress,
		networkParams.TotalTerminalDifficulty,
	)
	require.NoError(t, err)

	_, err = dataGeneratorCtx.GenerateCLGenesisData(
		genesisConfigTemplate,
		genesisMnemonicsTemplate,
		elGenesisData,
		uint64(time.Now().Unix()),
		networkParams.SecondsPerSlot,
		networkParams.AltairForkEpoch,
		networkParams.MergeForkEpoch,
		uint32(len(participantParams)),
		networkParams.NumValidatorKeysPerNode,
	)
	require.NoError(t, err)
}