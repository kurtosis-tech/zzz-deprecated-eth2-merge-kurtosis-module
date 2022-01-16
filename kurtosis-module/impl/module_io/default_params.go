package module_io

func getDefaultParams() *ExecuteParams {
	return &ExecuteParams{
		Participants: []*ParticipantParams{
			{
				ELClientType: ParticipantELClientType_Geth,
				CLClientType: ParticipantCLClientType_Nimbus,
			},
		},
		Network:             &NetworkParams{
			NetworkID:                          "3151908",
			DepositContractAddress:             "0x4242424242424242424242424242424242424242",
			SecondsPerSlot:                     12,
			SlotsPerEpoch:                      32,
			AltairForkEpoch:                    1,
			MergeForkEpoch:                     2,
			TotalTerminalDifficulty:            60000000,
			NumValidatorKeysPerNode:            32,
			PreregisteredValidatorKeysMnemonic: "giant issue aisle success illegal bike spike question tent bar rely arctic volcano long crawl hungry vocal artwork sniff fantasy very lucky have athlete",
		},
		WaitForFinalization: false,
		ClientLogLevel:      ParticipantLogLevel_Info,
	}
}
