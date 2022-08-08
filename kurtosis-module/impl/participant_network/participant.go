package participant_network

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/mev_boost"
)

// NOTE: We saw 1 Geth node + 3 Teku nodes causing problems, and the Teku folks
//  let us know that generally each CL node should be paired with 1 EL node
// https://discord.com/channels/697535391594446898/697539289042649190/922266717667856424
// We use this Participant class to represent a participant in the network who is running 1 of each type of client
type Participant struct {
	elClientType module_io.ParticipantELClientType
	clClientType module_io.ParticipantCLClientType

	elClientContext *el.ELClientContext
	clClientContext *cl.CLClientContext
	mevBoostContext *mev_boost.MEVBoostContext
}

func NewParticipant(elClientType module_io.ParticipantELClientType, clClientType module_io.ParticipantCLClientType, elClientContext *el.ELClientContext, clClientContext *cl.CLClientContext, mevBoostContext *mev_boost.MEVBoostContext) *Participant {
	return &Participant{elClientType: elClientType, clClientType: clClientType, elClientContext: elClientContext, clClientContext: clClientContext, mevBoostContext: mevBoostContext}
}

func (participant *Participant) GetELClientType() module_io.ParticipantELClientType {
	return participant.elClientType
}
func (participant *Participant) GetCLClientType() module_io.ParticipantCLClientType {
	return participant.clClientType
}
func (participant *Participant) GetELClientContext() *el.ELClientContext {
	return participant.elClientContext
}
func (participant *Participant) GetCLClientContext() *cl.CLClientContext {
	return participant.clClientContext
}
