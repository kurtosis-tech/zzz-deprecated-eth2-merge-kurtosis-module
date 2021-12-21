package participant_network

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
)

// NOTE: We saw 1 Geth node + 3 Teku nodes causing problems, and the Teku folks
//  let us know that generally each CL node should be paired with 1 EL node
// https://discord.com/channels/697535391594446898/697539289042649190/922266717667856424
// We use this Participant class to represent a participant in the network who is running 1 of each type of client
type Participant struct {
	elClientType ParticipantELClientType
	clClientType ParticipantCLClientType

	elClientContext *el.ELClientContext
	clClientContext *cl.CLClientContext
}

func NewParticipant(elClientType ParticipantELClientType, clClientType ParticipantCLClientType, elClientContext *el.ELClientContext, clClientContext *cl.CLClientContext) *Participant {
	return &Participant{elClientType: elClientType, clClientType: clClientType, elClientContext: elClientContext, clClientContext: clClientContext}
}

func (participant *Participant) GetELClientType() ParticipantELClientType {
	return participant.elClientType
}
func (participant *Participant) GetCLClientType() ParticipantCLClientType {
	return participant.clClientType
}
func (participant *Participant) GetELClientContext() *el.ELClientContext {
	return participant.elClientContext
}
func (participant *Participant) GetCLClientContext() *cl.CLClientContext {
	return participant.clClientContext
}
