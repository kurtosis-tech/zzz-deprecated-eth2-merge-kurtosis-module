package module_io

import (
	"encoding/json"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"strings"
)

const (
	expectedSecondsPerSlot = 12
	expectedSlotsPerEpoch = 32

	// TODO Remove this once Teku fixes its bug with merge fork epoch:
	//  https://discord.com/channels/697535391594446898/697539289042649190/935029250858299412
	tekuMinimumMergeForkEpoch = 3
)

func DeserializeAndValidateParams(paramsStr string) (*ExecuteParams, error) {
	paramsObj := GetDefaultExecuteParams()
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
			validClientTypes := []string{}
			for clientType := range validParticipantELClientTypes {
				validClientTypes = append(validClientTypes, string(clientType))
			}
			return nil, stacktrace.NewError(
				"Participant %v declares unrecognized EL client type '%v'; valid values are: %v",
				idx,
				elClientType,
				strings.Join(validClientTypes, ", "),
			 )
		}
		if participant.ELClientImage == useDefaultElImageKeyword {
			defaultElClientImage, found := defaultElImages[elClientType]
			if !found {
				return nil, stacktrace.NewError("EL client image wasn't provided, and no default image was defined for EL client type '%v'; this is a bug in the module", elClientType)
			}
			// Go's "range" is by-value, so we need to actually by-reference modify the paramsObj we need to
			//  use the idx
			paramsObj.Participants[idx].ELClientImage = defaultElClientImage
		}

		clClientType := participant.CLClientType
		if _, found := validParticipantCLClientTypes[clClientType]; !found {
			validClientTypes := []string{}
			for clientType := range validParticipantCLClientTypes {
				validClientTypes = append(validClientTypes, string(clientType))
			}
			return nil, stacktrace.NewError(
				"Participant %v declares unrecognized CL client type '%v'; valid values are: %v",
				idx,
				clClientType,
				strings.Join(validClientTypes, ", "),
			 )
		}
		if participant.CLClientImage == useDefaultClImageKeyword {
			defaultClClientImage, found := defaultClImages[clClientType]
			if !found {
				return nil, stacktrace.NewError("CL client image wasn't provided, and no default image was defined for CL client type '%v'; this is a bug in the module", clClientType)
			}
			// Go's "range" is by-value, so we need to actually by-reference modify the paramsObj we need to
			//  use the idx
			paramsObj.Participants[idx].CLClientImage = defaultClClientImage
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
	if networkParams.SecondsPerSlot != expectedSecondsPerSlot {
		logrus.Warnf("The current seconds-per-slot value is set to '%v'; values that aren't '%v' may cause the network to behave strangely", networkParams.SecondsPerSlot, expectedSecondsPerSlot)
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


	// TODO Remove this once we have a way of getting a Prysm node's ENR before genesis time:
	//  https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/issues/37
	if paramsObj.Participants[0].CLClientType == ParticipantCLClientType_Prysm {
		return nil, stacktrace.NewError("Cannot have a Prysm node as the boot CL node due to https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/issues/37")
	}

	// TODO Remove this check once Teku fixes its bug! See:
	//  https://discord.com/channels/697535391594446898/697539289042649190/935029250858299412
	hasTeku := false
	for _, participant := range paramsObj.Participants {
		hasTeku = hasTeku || participant.CLClientType == ParticipantCLClientType_Teku
	}
	if hasTeku && networkParams.MergeForkEpoch < tekuMinimumMergeForkEpoch {
		return nil, stacktrace.NewError(
			"Merge fork epoch is '%v' but cannot be < %v when a Teku node is present due to the following bug in Teku: https://discord.com/channels/697535391594446898/697539289042649190/935029250858299412",
			networkParams.MergeForkEpoch,
			tekuMinimumMergeForkEpoch,
		)
	}

	return paramsObj, nil
}
