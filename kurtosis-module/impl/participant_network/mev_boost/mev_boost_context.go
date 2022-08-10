package mev_boost

import (
	"fmt"
)

type MEVBoostContext struct {
	privateIPAddress string
	port             uint16
}

func (ctx *MEVBoostContext) Endpoint() string {
	return fmt.Sprintf("http://%s:%d", ctx.privateIPAddress, ctx.port)
}
