package module_io

import "github.com/kurtosis-tech/stacktrace"

const (
	// If these values are provided for the EL/CL images, then the client type-specific default image will be used
	useDefaultElImageKeyword = ""
	useDefaultClImageKeyword = ""

	unspecifiedLogLevel                 = ""
)

var defaultElImages = map[ParticipantELClientType]string{
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	ParticipantELClientType_Geth:       "parithoshj/geth:merge-371a525", // From around 2022-03-03
	ParticipantELClientType_Nethermind: "nethermindeth/nethermind:kintsugi_0.5",
	ParticipantELClientType_Besu:       "hyperledger/besu:merge",
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
}

var defaultClImages = map[ParticipantCLClientType]string{
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	ParticipantCLClientType_Lighthouse: "sigp/lighthouse:latest-unstable",
	ParticipantCLClientType_Teku:       "consensys/teku:latest",
	ParticipantCLClientType_Nimbus:     "statusim/nimbus-eth2:amd64-latest",
	// NOTE: Prysm actually has two images - a Beacon and a validator - so we pass in a comma-separated
	//  "beacon_image,validator_image" string
	ParticipantCLClientType_Prysm:    "prysmaticlabs/prysm-beacon-chain:latest,prysmaticlabs/prysm-validator:latest",
	ParticipantCLClientType_Lodestar: "chainsafe/lodestar:next",
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
}

// To see the exact JSON keys needed to override these values, see the ExecuteParams object and look for the
//  `json:"XXXXXXX"` metadata on the ExecuteParams properties
func GetDefaultExecuteParams() *ExecuteParams {
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	return &ExecuteParams{
		Participants: []*ParticipantParams{
			{
				ELClientType:     ParticipantELClientType_Geth,
				ELClientImage:    useDefaultElImageKeyword,
				ELClientLogLevel: unspecifiedLogLevel,
				CLClientType:     ParticipantCLClientType_Lighthouse,
				CLClientImage:    useDefaultClImageKeyword,
				CLClientLogLevel: unspecifiedLogLevel,
			},
		},
		Network: &NetworkParams{
			NetworkID:                          "3151908",
			DepositContractAddress:             "0x4242424242424242424242424242424242424242",
			ClGenesisTimeAdditionalDelay:       0,
			ElGenesisTimeAdditionalDelay:       0,
			SecondsPerSlot:                     12,
			SlotsPerEpoch:                      32,
			AltairForkEpoch:                    1,
			MergeForkEpoch:                     2,
			TotalTerminalDifficulty:            100000000,
			NumValidatorKeysPerNode:            64,
			NumExtraValidatorKeys: 			    0,
			PreregisteredValidatorKeysMnemonic: "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete",
		},
		WaitForMining:       true,
		WaitForFinalization: false,
		WaitForClGenesis:    true,
		ClientLogLevel:      GlobalClientLogLevel_Info,
	}
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//       If you change these in any way, modify the example JSON config in the README to reflect this!
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
}

// Gets the string of the log level that the client should log at:
//  - If the participant-specific log level string is present, use that
//  - If the participant-specific log level string is empty, use the global default
func GetClientLogLevelStrOrDefault(participantLogLevel string, globalLogLevel GlobalClientLogLevel, clientLogLevels map[GlobalClientLogLevel]string) (string, error) {

	var (
		logLevel = participantLogLevel
		found bool
	)

	if logLevel == unspecifiedLogLevel {
		logLevel, found = clientLogLevels[globalLogLevel]
		if !found {
			return "", stacktrace.NewError("No participant log level defined for global client log level '%v'; this is a bug in the module", globalLogLevel)
		}
	}

	return logLevel, nil
}
