package el

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"strconv"
	"text/template"
)

const (
	nethermindGenesisJsonFilename = "nethermind.json"
)

type nethermindGenesisJsonTemplateData struct {
	NetworkIDAsHex string
	// TODO add genesis timestamp here???
}

func generateNethermindGenesis(
	genesisJsonTemplate *template.Template,
	networkId string,
	genesisUnixTimestamp uint64,
	totalTerminalDifficulty uint64,
	serviceCtx *services.ServiceContext,
	outputSharedDir *services.SharedPath,
) (
	string,
	error,
) {
	networkIdAsHex, err := getNetworkIdHexSting(networkId)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred rendering network ID '%v' as a hex string",
			networkId,
		)
	}

	templateData := nethermindGenesisJsonTemplateData{
		NetworkIDAsHex: networkIdAsHex,
	}
	genesisJsonSharedFile := outputSharedDir.GetChildPath(nethermindGenesisJsonFilename)
	if err := service_launch_utils.FillTemplateToSharedPath(genesisJsonTemplate, templateData, genesisJsonSharedFile); err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error generating the Nethermind genesis JSON from template",
		 )
	}

	return genesisJsonSharedFile.GetAbsPathOnThisContainer(), nil
}

func getNetworkIdHexSting(networkId string) (string, error) {
	uintBase := 10
	uintBits := 64
	networkIdUint64, err := strconv.ParseUint(networkId, uintBase, uintBits)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred parsing network ID string '%v' to uint with base %v and %v bits",
			networkId,
			uintBase,
			uintBits,
		)
	}
	return fmt.Sprintf("0x%x", networkIdUint64), nil
}
