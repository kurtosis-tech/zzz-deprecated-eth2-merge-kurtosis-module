package participant_network

import "sync"

// Represents a network of virtual "participants", where each participant runs:
//  1) an EL client
//  2) a Beacon client
//  3) a validator client
type ParticipantNetwork struct {
	participants map[uint32]*Participant

	mutex *sync.Mutex
}

// TODO constructor

func AddParticipant(elClientType ParticipantELClientType, clClientType ParticipantCLClientType) error {
	// Get the launchers for the appropriate EL & CL

	// Launch the EL using the launcher

	// Launch the CL using the launcher & EL information
}