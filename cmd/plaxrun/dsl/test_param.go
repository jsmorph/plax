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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	plaxDsl "github.com/Comcast/plax/dsl"
)

// TestParamEnvMap type
type TestParamEnvMap map[string]string

// TestParamDependency name
type TestParamDependency string

// TestParamDependencyList parameter dependency list
type TestParamDependencyList []TestParamDependency

// process the TestParamDependencyList
func (tpdl TestParamDependencyList) process(ctx *plaxDsl.Ctx, tpbm TestParamBindingMap, bs *plaxDsl.Bindings) error {
	for _, tpd := range tpdl {
		err := tpd.process(ctx, tpbm, bs)
		if err != nil {
			return fmt.Errorf("failed to process test params: %w", err)
		}
	}

	return nil
}

// TestParamMap of parameter keys to non substituted parameter values
type TestParamMap map[string]string

// bind the TestParams from the TestParamMap to the bs Bindings
func (tpm TestParamMap) bind(ctx *plaxDsl.Ctx, bs *plaxDsl.Bindings) error {
	for tpk, tpv := range tpm {
		pv, err := bs.StringSub(ctx, tpv)
		if err != nil {
			return fmt.Errorf("failed to substitute test parameter: %w", err)
		}
		bs.SetKeyValue(tpk, pv)
	}

	return nil
}

// process the TestParamDependency
func (tpd TestParamDependency) process(ctx *plaxDsl.Ctx, tpbm TestParamBindingMap, bs *plaxDsl.Bindings) error {
	tpk := string(tpd)
	pbm, ok := tpbm[string(tpk)]
	if !ok {
		return fmt.Errorf("failed to find test param %s", tpk)
	}

	for _, tpd := range pbm.DependsOn {
		if err := tpd.process(ctx, tpbm, bs); err != nil {
			return fmt.Errorf("failed to process dependent param for %s: %v", tpk, err)
		}
	}

	err := pbm.process(ctx, tpk, bs)
	if err != nil {
		return fmt.Errorf("failed to process param %s: %v", tpk, err)
	}

	return nil
}

// TestParamBindingMap is a map of TestParamBinding
type TestParamBindingMap map[string]TestParamBinding

// TestParamBinding binds a value to a parameter via a command
type TestParamBinding struct {
	// DependsOn is a list of dependent parameters
	DependsOn TestParamDependencyList `json:"dependsOn" yaml:"dependsOn"`

	// Cmd is the commmand name of the program.
	//
	// Subject to expansion.
	Cmd string `json:"cmd" yaml:"cmd"`

	// Args is the list of command-line arguments for the program.
	//
	// Subject to expansion.
	Args []string `json:"args" yaml:"args"`

	// cmd is the private exec command
	ec *exec.Cmd

	// tpem is the map of environemnt variables to pass into the Run script
	Envs TestParamEnvMap `json:"envs" yaml:"envs"`
}

// environment set the environment fo the script execution
func (tpb *TestParamBinding) environment(ctx *plaxDsl.Ctx, key string, bs *plaxDsl.Bindings) error {
	tpb.ec.Env = os.Environ()

	tpb.ec.Env = append(tpb.ec.Env, fmt.Sprintf("KEY=%s", key))

	for k, v := range tpb.Envs {
		val, err := bs.StringSub(ctx, v)
		if err != nil {
			return fmt.Errorf("failed to bind envs for key=%s: %w", k, err)
		}
		ctx.Logdf("binding envs %s=%s", k, val)
		tpb.ec.Env = append(tpb.ec.Env, fmt.Sprintf("%s=%s", k, val))
	}

	return nil
}

// substitute performs a string substitution of the enviornment variables with the Bindings
func (tpb *TestParamBinding) substitute(ctx *plaxDsl.Ctx, bs *plaxDsl.Bindings) (*TestParamBinding, error) {
	// Substitute the environment variable bindings
	tpem := make(TestParamEnvMap)

	for k, v := range tpb.Envs {
		s, err := bs.StringSub(ctx, v)
		if err != nil {
			return nil, err
		}
		tpem[k] = s
	}

	return &TestParamBinding{
		DependsOn: tpb.DependsOn,
		Cmd:       tpb.Cmd,
		Args:      tpb.Args,
		Envs:      tpem,
		ec:        tpb.ec,
	}, nil
}

// run the command to process parameter binding
func (tpb *TestParamBinding) run(ctx *plaxDsl.Ctx, key string, bs *plaxDsl.Bindings) error {
	var err error

	// Substitute the parameter and run command bindings
	tpb, err = tpb.substitute(ctx, bs)
	if err != nil {
		return err
	}

	// Build the execution command
	tpb.ec = exec.Command(tpb.Cmd, tpb.Args...)

	// Setup the environment with the substitute parameters
	if err := tpb.environment(ctx, key, bs); err != nil {
		return err
	}

	tpb.ec.Stderr = os.Stderr
	tpb.ec.Stdin = os.Stdin
	var stdout bytes.Buffer
	tpb.ec.Stdout = &stdout

	if err := tpb.ec.Run(); err != nil {
		ctx.Logdf("Param binding command %s error on run: %s\n", key, err)
		return err
	}

	if tpb.ec.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("Param binding process termintated with exit code %d", tpb.ec.ProcessState.ExitCode())
	}

	output := strings.TrimSuffix(stdout.String(), "\n") // removing only the trailing newline

	values := strings.Split(output, "\n")
	for _, value := range values {
		ctx.Logdf("Binding %s", value)
		bs.Set(value)
	}

	return nil
}

// Process the test param binding
func (tpb *TestParamBinding) process(ctx *plaxDsl.Ctx, pk string, bs *plaxDsl.Bindings) error {
	// If paramater binding already exists just return
	if _, ok := (*bs)[pk]; ok {
		return nil
	}

	// Process the parameter binding run command
	if err := tpb.run(ctx, pk, bs); err != nil {
		return err
	}

	return nil
}
