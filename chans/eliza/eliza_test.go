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
	"testing"
	"time"

	"github.com/Comcast/plax/dsl"
)

func TestDocs(t *testing.T) {
	(&Eliza{}).DocSpec().Write("eliza")
}

func TestEliza(t *testing.T) {
	ctx := dsl.NewCtx(nil)

	c, err := NewEliza(ctx, nil)
	if err != nil {
		t.Fatal("could not create cmd channel: " + err.Error())
	}

	if c.Kind() != "eliza" {
		t.Fatal(c.Kind())
	}

	if err = c.Open(ctx); err != nil {
		t.Fatal(err)
	}

	msg := dsl.Msg{
		Payload:    "I sure do like queso",
		ReceivedAt: time.Now(),
	}
	if err = c.Pub(ctx, msg); err != nil {
		t.Fatal(err)
	}

	in := c.Recv(ctx)

	msg = <-in

	// log.Printf("%#v", msg)
}
