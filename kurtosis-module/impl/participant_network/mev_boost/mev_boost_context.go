package mev_boost

import (
	"fmt"

	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type MEVBoostContext struct {
	service *services.ServiceContext
}

func (ctx *MEVBoostContext) Endpoint() string {
	ports := ctx.service.GetPrivatePorts()
	port, ok := ports["api"]
	if !ok {
		panic("invariant violated, check port handling")
	}
	return fmt.Sprintf("http://%s:%d", ctx.service.GetPrivateIPAddress(), port)
}
