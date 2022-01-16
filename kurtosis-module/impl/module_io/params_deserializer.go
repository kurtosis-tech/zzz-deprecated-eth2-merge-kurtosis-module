package module_io

import (
	"encoding/json"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"strings"
)

const (
	expectedSlotsPerEpoch = 32
)

func DeserializeAndValidateParams(paramsStr string) (*ExecuteParams, error) {
	paramsObj := getDefaultParams()
	if err := json.Unmarshal([]byte(paramsStr), paramsObj); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred deserializing the serialized params")
	}

	if _, found := validParticipantLogLevels[paramsObj.ClientLogLevel]; !found {
		return nil, stacktrace.NewError("Unrecognized client log level '%v'", paramsObj.ClientLogLevel)
	}

	// Validate participants
	if len(paramsObj.Participants) == 0 {
		return nil, stacktrace.NewError("At least one participant is required")
	}
	for idx, participant := range paramsObj.Participants {
		if idx == 0 && participant.ELClientType == ParticipantELClientType_Nethermind {
			return nil, stacktrace.NewError("Cannot use a Nethermind client for the first participant because Nethermind clients don't mine on Eth1")
		}

		elClientType := participant.ELClientType
		if _, found := validParticipantELClientTypes[elClientType]; !found {
			return nil, stacktrace.NewError("Participant %v declares unrecognized EL client type '%v'", idx, elClientType)
		}

		clClientType := participant.CLClientType
		if _, found := validParticipantCLClientTypes[clClientType]; !found {
			return nil, stacktrace.NewError("Participant %v declares unrecognized CL client type '%v'", idx, clClientType)
		}
	}

	networkParams := paramsObj.Network
	if len(strings.TrimSpace(networkParams.NetworkID)) == 0 {
		return nil, stacktrace.NewError("Network ID must not be empty")
	}
	if len(strings.TrimSpace(networkParams.DepositContractAddress)) == 0 {
		return nil, stacktrace.NewError("Deposit contract address must not be empty")
	}

	// Slot/epoch validation
	if networkParams.SecondsPerSlot == 0 {
		return nil, stacktrace.NewError("Each slot must be >= 1 second")
	}
	if networkParams.SlotsPerEpoch == 0 {
		return nil, stacktrace.NewError("Each epoch must be composed of >= 1 slot")
	}
	if networkParams.SlotsPerEpoch != expectedSlotsPerEpoch {
		logrus.Warnf("The current slots-per-epoch value is set to '%v'; values that aren't '%v' may cause the network to behave strangely", networkParams.SlotsPerEpoch, expectedSlotsPerEpoch)
	}

	// Fork epoch validation
	if networkParams.AltairForkEpoch == 0 {
		return nil, stacktrace.NewError("Altair fork epoch must be >= 1")
	}
	if networkParams.MergeForkEpoch == 0 {
		return nil, stacktrace.NewError("Merge fork epoch must be >= 1")
	}
	if networkParams.MergeForkEpoch <= networkParams.AltairForkEpoch {
		return nil, stacktrace.NewError("Altair fork epoch must be < merge fork epoch")
	}

	if networkParams.TotalTerminalDifficulty == 0 {
		return nil, stacktrace.NewError("Total terminal difficulty must be >= 1")
	}
	// TODO validation to ensure TTD comes after merge fork epoch

	// Validator validation
	requiredNumValidators := 2 * networkParams.SlotsPerEpoch
	actualNumValidators := uint32(len(paramsObj.Participants)) * networkParams.NumValidatorKeysPerNode
	if actualNumValidators < requiredNumValidators {
		return nil, stacktrace.NewError(
			"We need %v validators (enough for two epochs, with one validator per slot), but only have %v",
			requiredNumValidators,
			actualNumValidators,
		 )
	}
	if len(strings.TrimSpace(networkParams.PreregisteredValidatorKeysMnemonic)) == 0 {
		return nil, stacktrace.NewError("Preregistered validator keys mnemonic must not be empty")
	}


	return paramsObj, nil
}
