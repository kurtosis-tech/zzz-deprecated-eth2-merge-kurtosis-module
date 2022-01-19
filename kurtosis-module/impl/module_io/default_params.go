package module_io

const (
	// If these values are provided for the EL/CL images, then the client type-specific default image will be used
	useDefaultElImageKeyword = ""
	useDefaultClImageKeyword = ""
)

var defaultElImages = map[ParticipantELClientType]string{
	ParticipantELClientType_Geth: "parithoshj/geth:merge-f72c361", // From around 2022-01-18
	ParticipantELClientType_Nethermind: "nethermindeth/nethermind:kintsugi_0.5",
}

var defaultClImages = map[ParticipantCLClientType]string{
	ParticipantCLClientType_Lighthouse: "sigp/lighthouse:latest-unstable",
	ParticipantCLClientType_Teku:       "consensys/teku:latest",
	ParticipantCLClientType_Nimbus:     "statusim/nimbus-eth2:amd64-latest",
	// NOTE: Prysm actually has two images - a Beacon and a validator - so we pass in a comma-separated
	//  "beacon_image,validator_image" string
	ParticipantCLClientType_Prysm:      "prysmaticlabs/prysm-beacon-chain:latest,prysmaticlabs/prysm-validator:latest",
	ParticipantCLClientType_Lodestar:   "chainsafe/lodestar:next",
}

// To see the exact JSON keys needed to override these values, see the ExecuteParams object and look for the
//  `json:"XXXXXXX"` metadata on the ExecuteParams properties
func GetDefaultExecuteParams() *ExecuteParams {
	return &ExecuteParams{
		Participants: []*ParticipantParams{
			{
				ELClientType:  ParticipantELClientType_Geth,
				ELClientImage: useDefaultElImageKeyword,
				CLClientType:  ParticipantCLClientType_Nimbus,
				CLClientImage: useDefaultClImageKeyword,
			},
		},
		Network: &NetworkParams{
			NetworkID:                          "3151908",
			DepositContractAddress:             "0x4242424242424242424242424242424242424242",
			SecondsPerSlot:                     12,
			SlotsPerEpoch:                      32,
			AltairForkEpoch:                    1,
			MergeForkEpoch:                     2,
			TotalTerminalDifficulty:            100000000,
			NumValidatorKeysPerNode:            64,
			PreregisteredValidatorKeysMnemonic: "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete",
		},
		WaitForMining:       true,
		WaitForFinalization: false,
		ClientLogLevel:      ParticipantLogLevel_Info,
	}
}