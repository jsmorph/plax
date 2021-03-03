package main

import (
	"log"

	"github.com/Comcast/plax/chans/eliza"
	"github.com/Comcast/plax/dsl"
)

func init() {
	log.Printf("registering Eliza from plugin main")
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "eliza", eliza.NewEliza)
}

func main() {
}
