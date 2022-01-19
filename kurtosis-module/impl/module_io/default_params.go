package module_io

var defaultElImages = map[ParticipantELClientType]string{
	ParticipantELClientType_Geth: "parithoshj/geth:merge-f72c361", // From around 2022-01-18
	ParticipantELClientType_Nethermind: "nethermindeth/nethermind:kintsugi_0.5",
}

var defaultClImage = map[ParticipantCLClientType]string{
	ParticipantCLClientType_Lighthouse: "sigp/lighthouse:latest-unstable",
	ParticipantCLClientType_Teku:       "consensys/teku:latest",
	ParticipantCLClientType_Nimbus:     "statusim/nimbus-eth2:amd64-latest",
	ParticipantCLClientType_Prysm:      true,
	ParticipantCLClientType_Lodestar:   true,
}

// To see the exact JSON keys needed to override these values, see the ExecuteParams object and look for the
//  `json:"XXXXXXX"` metadata on the ExecuteParams properties
func GetDefaultExecuteParams() *ExecuteParams {
	return &ExecuteParams{
		Participants: []*ParticipantParams{
			{
				ELClientType: ParticipantELClientType_Geth,
				CLClientType: ParticipantCLClientType_Nimbus,
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
