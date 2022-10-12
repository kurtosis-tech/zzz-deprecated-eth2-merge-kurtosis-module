package impl

import (
	"context"
	"encoding/json"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/genesis_consts"
	"io/ioutil"
	"time"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/forkmon"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/grafana"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prometheus"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/transaction_spammer"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
)

const (
	responseJsonLinePrefixStr = ""
	responseJsonLineIndentStr = "  "

	// On mainnet, finalization will be head - 2
	// However, according to Pari, on these small testnets with genesis very close there's more churn so 4 epochs is possible
	firstHeadEpochWhereFinalizedEpochIsPossible = uint64(4)
	// The number of extra epochs beyond the first-epoch-where-finalization-is-possible that we'll wait for the network to finalize
	finalizedEpochTolerance         = uint64(0)
	timeBetweenFinalizedEpochChecks = 5 * time.Second

	grafanaUser             = "admin"
	grafanaPassword         = "admin"
	grafanaDashboardPathUrl = "/d/QdTOwy-nz/eth2-merge-kurtosis-module-dashboard?orgId=1"
)

type Eth2KurtosisModule struct {
}

func NewEth2KurtosisModule() *Eth2KurtosisModule {
	return &Eth2KurtosisModule{}
}

func (e Eth2KurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	ctx := context.Background()

	logrus.Infof("Deserializing the following execute params:\n%v", serializedParams)
	paramsObj, err := module_io.DeserializeAndValidateParams(serializedParams)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred deserializing & validating the params")
	}
	networkParams := paramsObj.Network
	numParticipants := uint32(len(paramsObj.Participants))
	logrus.Info("Successfully deserialized execute params")

	// Parse templates early, so that any errors are caught before we do the stuff that takes a long time
	grafanaDatasourceConfigTemplate, err := ioutil.ReadFile(static_files.GrafanaDatasourceConfigTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred reading Grafana datasource config template file '%v'", static_files.PrometheusConfigTemplateFilepath)
	}
	grafanaDashboardsConfigTemplate, err := ioutil.ReadFile(static_files.GrafanaDashboardProvidersConfigTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred reading Grafana dashboards config template file '%v'", static_files.GrafanaDashboardProvidersConfigTemplateFilepath)
	}
	prometheusConfigTemplate, err := ioutil.ReadFile(static_files.PrometheusConfigTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred reading prometheus config template file '%v'", static_files.PrometheusConfigTemplateFilepath)
	}

	grafanaDatasourceConfigTemplateString := string(grafanaDatasourceConfigTemplate)
	grafanaDashboardsConfigTemplateString := string(grafanaDashboardsConfigTemplate)
	prometheusConfigTemplateString := string(prometheusConfigTemplate)

	logrus.Infof("Adding %v participants logging at level '%v'...", numParticipants, paramsObj.ClientLogLevel)
	participants, clGenesisUnixTimestamp, err := participant_network.LaunchParticipantNetwork(
		ctx,
		enclaveCtx,
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

	// TODO This is a temporary hack to only starts the Ethereum network until the product supports easily decomposing this module
	if !paramsObj.LaunchAdditionalServices {
		return "{}", nil
	}

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

	logrus.Info("Waiting until CL genesis occurs to add forkmon...")
	// We need to wait until the CL genesis has been reached to launch Forkmon because it has a bug (as of 2022-01-18) where
	//  if a CL ndoe's getHealth endpoint returns a non-200 error code, Forkmon will mark the node as failed and will never revisit it
	// This is fine with nodes who report 200 before genesis, but certain nodes (e.g. Lighthouse) will report a 503 before genesis
	// Therefore, the simple fix is wait until CL genesis to start Forkmon
	secondsRemainingUntilClGenesis := clGenesisUnixTimestamp - uint64(time.Now().Unix())
	if secondsRemainingUntilClGenesis < 0 {
		secondsRemainingUntilClGenesis = 0
	}
	durationUntilClGenesis := time.Duration(int64(secondsRemainingUntilClGenesis)) * time.Second
	time.Sleep(durationUntilClGenesis)
	logrus.Info("CL genesis has occurred")

	logrus.Info("Launching forkmon...")
	forkmonConfigTemplate, err := ioutil.ReadFile(static_files.ForkmonConfigTemplateFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred reading forkmon config template file '%v'", static_files.ForkmonConfigTemplateFilepath)
	}
	forkmonConfigTemplateString := string(forkmonConfigTemplate)
	err = forkmon.LaunchForkmon(
		enclaveCtx,
		forkmonConfigTemplateString,
		allClClientContexts,
		clGenesisUnixTimestamp,
		networkParams.SecondsPerSlot,
		networkParams.SlotsPerEpoch,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching forkmon service")
	}
	logrus.Infof("Successfully launched forkmon")

	logrus.Info("Launching prometheus...")
	prometheusPrivateUrl, err := prometheus.LaunchPrometheus(
		enclaveCtx,
		prometheusConfigTemplateString,
		allClClientContexts,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching prometheus service")
	}
	logrus.Infof("Successfully launched Prometheus")

	logrus.Info("Launching grafana...")
	err = grafana.LaunchGrafana(
		enclaveCtx,
		grafanaDatasourceConfigTemplateString,
		grafanaDashboardsConfigTemplateString,
		prometheusPrivateUrl,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching Grafana")
	}
	logrus.Infof("Successfully launched Grafana. The eth2 merge module dashboard can be reached via path '%v'", grafanaDashboardPathUrl)

	responseObj := &module_io.ExecuteResponse{
		GrafanaInfo: &module_io.GrafanaInfo{
			DashboardPath: grafanaDashboardPathUrl,
			User:          grafanaUser,
			Password:      grafanaPassword,
		},
	}
	responseStr, err := json.MarshalIndent(responseObj, responseJsonLinePrefixStr, responseJsonLineIndentStr)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred serializing the following response object to JSON for returning: %+v", responseObj)
	}

	return string(responseStr), nil
}
