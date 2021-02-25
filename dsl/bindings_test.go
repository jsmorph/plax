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

package dsl

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

// TestBindingsValue test the Bindings methods for implementing
// flag.Value.
func TestBindingsValue(t *testing.T) {
	bs := NewBindings()

	bs.Set(`want="queso"`)
	bs.Set(`like=tacos`)
	bs.String()

	x, have := bs["want"]
	if !have {
		t.Fatal("lost 'want'")
	}
	s, is := x.(string)
	if !is {
		t.Fatalf("x is a %T", x)
	}
	if s != "queso" {
		t.Fatal(s)
	}
}

func TestBindingsCopy(t *testing.T) {
	bs0 := NewBindings()
	bs0["want"] = "queso"
	bs1, err := bs0.Copy()
	if err != nil {
		t.Fatal(err)
	}
	(*bs1)["want"] = "tacos"

	{
		x, have := bs0["want"]
		if !have {
			t.Fatal("lost 'want'")
		}
		s, is := x.(string)
		if !is {
			t.Fatalf("x is a %T", x)
		}
		if s != "queso" {
			t.Fatal(s)
		}
	}

	{
		x, have := (*bs1)["want"]
		if !have {
			t.Fatal("lost 'want'")
		}
		s, is := x.(string)
		if !is {
			t.Fatalf("x is a %T", x)
		}
		if s != "tacos" {
			t.Fatal(s)
		}
	}
}

func TestSubstitute(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		tst = NewTest(ctx, "a", nil)
	)

	t.Run("basic", func(t *testing.T) {
		// We bind variables that require recursive
		// subsitution.  Note that these variables (by
		// definition) start with '?'.
		tst.Bindings = map[string]interface{}{
			"?want":  "{?queso}",
			"?queso": "queso",
		}
		s, err := tst.Bindings.StringSub(ctx, `!!"I want " + "{?want}."`)
		if err != nil {
			t.Fatal(err)
		}
		if s != "I want queso." {
			t.Fatal(s)
		}
	})

	t.Run("constantEmbedded", func(t *testing.T) {
		// Same basic test but we using a "binding" for a
		// constant (without the '?' prefix).
		tst.Bindings = map[string]interface{}{
			// Bind 'want' to a string that itself
			// references a binding variable.
			"want": "{?queso}",
			// Bind the variable referenced above.
			"?queso": "queso",
		}
		s, err := tst.Bindings.StringSub(ctx, `!!"I want " + "{want}."`)
		if err != nil {
			t.Fatal(err)
		}
		if s != "I want queso." {
			t.Fatal(s)
		}
	})

	t.Run("constantStructured", func(t *testing.T) {
		// Parameter-like subsitution: Bind a "parameter",
		// which has no "?" but does have "{}".
		tst.Bindings = map[string]interface{}{
			// Bind 'want' to a string that itself
			// references a binding variable.
			"{want}": "{?this}",
			// Bind the variable referenced above.
			"{?this}": "queso",
		}
		x := MaybeParseJSON(`{"need":"{want}"}`)
		var y interface{}
		if err := tst.Bindings.SubX(ctx, x, &y); err != nil {
			t.Fatal(err)
		}
		log.Printf("DEBUG y %s", JSON(y))
		js1, err := json.Marshal(&x)
		if err != nil {
			t.Fatal(err)
		}
		js2, err := json.Marshal(&y)
		if err != nil {
			t.Fatal(err)
		}
		if string(js1) != string(js2) {
			t.Fatal(string(js2))
		}
	})

	t.Run("deepstring", func(t *testing.T) {

		var (
			src, target struct {
				Foo struct {
					Bar string
				}
			}
			js = `{"Foo":{"Bar":"I want {?want}."}}`
		)

		if err := json.Unmarshal([]byte(js), &src); err != nil {
			t.Fatal(err)
		}

		tst.Bindings = map[string]interface{}{
			"?want": "queso",
		}
		if err := tst.Bindings.SubX(ctx, src, &target); err != nil {
			t.Fatal(err)
		}
		if s := target.Foo.Bar; s != "I want queso." {
			t.Fatal(s)
		}
	})

}

func TestSubstituteOnce(t *testing.T) {
	var (
		ctx = NewCtx(context.Background())
		tst = NewTest(ctx, "a", nil)
	)

	t.Run("badjs", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, "!!nope"); err == nil {
			t.Fatal("should have complained")
		}
	})

	t.Run("jsobj", func(t *testing.T) {
		if s, err := tst.Bindings.StringSubOnce(ctx, `!!({"want":"tacos"})`); err != nil {
			t.Fatal(err)
		} else if s != `{"want":"tacos"}` {
			t.Fatal(s)
		}
	})

	t.Run("jsnots", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, `!!function() {}`); err == nil {
			t.Fatal("should have complained")
		}
	})

	t.Run("file", func(t *testing.T) {
		if s, err := tst.Bindings.StringSubOnce(ctx, `@@test_test.go`); err != nil {
			t.Fatal(err)
		} else if len(s) < 1000 {
			t.Fatal(len(s))
		}
	})

	t.Run("filebad", func(t *testing.T) {
		if _, err := tst.Bindings.StringSubOnce(ctx, `@@nope`); err == nil {
			t.Fatal("should have complained")
		}
	})

	tst.Bindings = map[string]interface{}{
		"?need": "chips",
	}

	t.Run("filegood", func(t *testing.T) {
		s, err := tst.Bindings.StringSubOnce(ctx, `@@test_test.go`)
		if err != nil {
			t.Fatal(err)
		}
		// The following comment should be substituted!
		//
		// {?need}
		if strings.Contains(s, "{?need}") {
			t.Fatal("?need")
		}
	})
}

func TestAtAtSub(t *testing.T) {
	var (
		x      = "Some Go source code: {@@bindings_test.go}"
		ctx    = NewCtx(context.Background())
		y, err = atAtSub(ctx, x)
	)

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(y, "TestAtAtSub") {
		n := len(y)
		if 80 < n {
			y = y[0:80] + "..."
		}
		t.Fatal(y)
	}
}
