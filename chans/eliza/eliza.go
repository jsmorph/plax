/*
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package eliza

import (
	"fmt"
	"log"
	"time"

	"github.com/Comcast/plax/dsl"

	eliza "github.com/kennysong/goeliza"
)

func init() {
	log.Printf("registering Eliza")
	dsl.TheChanRegistry.Register(dsl.NewCtx(nil), "eliza", NewEliza)
}

type Eliza struct {
	c chan dsl.Msg
}

func NewEliza(ctx *dsl.Ctx, cfg interface{}) (dsl.Chan, error) {
	return &Eliza{
		c: make(chan dsl.Msg, 1024),
	}, nil
}

func (c *Eliza) DocSpec() *dsl.DocSpec {
	return &dsl.DocSpec{
		Chan: &Eliza{},
	}
}

func (c *Eliza) Kind() dsl.ChanKind {
	return "eliza"
}

func (c *Eliza) Open(ctx *dsl.Ctx) error {
	return nil
}

func (c *Eliza) Close(ctx *dsl.Ctx) error {
	return nil
}

func (c *Eliza) Sub(ctx *dsl.Ctx, topic string) error {
	return nil
}

func (c *Eliza) Pub(ctx *dsl.Ctx, m dsl.Msg) error {
	reply := eliza.ReplyTo(m.Payload)
	go func() {
		select {
		case <-ctx.Done():
		case c.c <- dsl.Msg{
			Payload: reply,
		}:
		}
	}()

	return nil
}

func (c *Eliza) Recv(ctx *dsl.Ctx) chan dsl.Msg {
	return c.c
}

func (c *Eliza) Kill(ctx *dsl.Ctx) error {
	return fmt.Errorf("you are not allowed to kill Eliza")
}

func (c *Eliza) To(ctx *dsl.Ctx, m dsl.Msg) error {
	m.ReceivedAt = time.Now().UTC()
	select {
	case <-ctx.Done():
	case c.c <- m:
	default:
		return fmt.Errorf("Eliza input buffer is full")
	}
	return nil
}
