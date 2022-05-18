/*
 * This file is part of the kiagnose project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package console

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"time"

	expect "github.com/google/goexpect"

	"google.golang.org/grpc/codes"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	kubevmi "github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/vmi"
)

const (
	PromptExpression = `(\$ |\# )`
	CRLF             = "\r\n"
)

type Console struct {
	client kubevmi.KubevirtVmisClient
	vmi    *v1.VirtualMachineInstance
}

func NewConsole(client kubevmi.KubevirtVmisClient, vmi *v1.VirtualMachineInstance) Console {
	return Console{client: client, vmi: vmi}
}

// LoginToFedora performs a console login to a Fedora base VM
func (c Console) LoginToFedora() error {
	const connectTimeout = 10 * time.Second
	expecter, err := c.newExpecter(connectTimeout)
	if err != nil {
		return err
	}
	defer expecter.Close()

	if e := expecter.Send("\n"); e != nil {
		return e
	}

	// Do not login, if we already logged in
	b := []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: fmt.Sprintf(`(\[fedora@(localhost|fedora|%s) ~\]\$ |\[root@(localhost|fedora|%s) fedora\]\# )`, c.vmi.Name, c.vmi.Name)},
	}
	const batchIsLoggedTimeout = 5 * time.Second
	if _, e := expecter.ExpectBatch(b, batchIsLoggedTimeout); e == nil {
		return nil
	}

	vmi := c.vmi

	b = []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				// and in case the VM's did not get hostname form DHCP server try the default hostname
				R:  regexp.MustCompile(fmt.Sprintf(`(localhost|fedora|%s) login: `, vmi.Name)),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Password:`),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Login incorrect`),
				T:  expect.LogContinue("Failed to log in", expect.NewStatus(codes.PermissionDenied, "login failed")),
				Rt: 10,
			},
			&expect.Case{
				R: regexp.MustCompile(fmt.Sprintf(`\[fedora@(localhost|fedora|%s) ~\]\$ `, vmi.Name)),
				T: expect.OK(),
			},
		}},
		&expect.BSnd{S: "sudo su\n"},
		&expect.BExp{R: PromptExpression},
	}
	const batchLoginTimeout = 2 * time.Minute
	res, err := expecter.ExpectBatch(b, batchLoginTimeout)
	if err != nil {
		log.Printf("Login attempt to VMI (%s) failed: %+v", c.vmi.Name, res)
		// Try once more since sometimes the login prompt is ripped apart by asynchronous daemon updates
		res, err := expecter.ExpectBatch(b, 1*time.Minute)
		if err != nil {
			log.Printf("Retried login attempt to VMI (%s) after two minutes failed: %+v", c.vmi.Name, res)
			return err
		}
	}

	return configureConsole(expecter)
}

// newExpecter will connect to an already logged in VMI console and return the generated expecter it will wait `timeout` for the connection.
func (c Console) newExpecter(timeout time.Duration, opts ...expect.Option) (expect.Expecter, error) {
	vmiReader, vmiWriter := io.Pipe()
	expecterReader, expecterWriter := io.Pipe()
	resCh := make(chan error)

	startTime := time.Now()
	con, err := c.client.SerialConsole(c.vmi.Namespace, c.vmi.Name, timeout)
	if err != nil {
		return nil, err
	}
	timeout -= time.Since(startTime)

	go func() {
		resCh <- con.Stream(kubecli.StreamOptions{
			In:  vmiReader,
			Out: expecterWriter,
		})
	}()

	opts = append(opts, expect.SendTimeout(timeout), expect.Verbose(true))
	expecter, _, err := expect.SpawnGeneric(&expect.GenOptions{
		In:  vmiWriter,
		Out: expecterReader,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			expecterWriter.Close()
			vmiReader.Close()
			return nil
		},
		Check: func() bool { return true },
	}, timeout, opts...)

	return expecter, err
}

func configureConsole(expecter expect.Expecter) error {
	batch := []expect.Batcher{
		&expect.BSnd{S: "stty cols 500 rows 500\n"},
		&expect.BExp{R: PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: RetValue("0")},
		&expect.BSnd{S: "dmesg -n 1\n"},
		&expect.BExp{R: PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: RetValue("0")},
	}
	const batchTimeout = 30 * time.Second
	resp, err := expecter.ExpectBatch(batch, batchTimeout)
	if err != nil {
		log.Printf("console configuration error: %+v", resp)
	}
	return err
}

func RetValue(retcode string) string {
	return "\n" + retcode + CRLF + ".*" + PromptExpression
}