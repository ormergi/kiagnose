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

package config_test

import (
	"fmt"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"

	"github.com/kiagnose/kiagnose/checkups/kubevirt-vm-latency/vmlatency/internal/config"
)

type configCreateTestCases struct {
	description    string
	env            map[string]string
	expectedConfig config.Config
}

func TestCreateConfigFromEnvShould(t *testing.T) {
	const (
		testNamespace           = "default"
		testResultConfigMapName = "result"
		testNetAttachDefName    = "blue-net"
		testDesiredMaxLatency   = time.Millisecond * 100
		testSampleDuration      = time.Minute * 1
	)

	testCases := []configCreateTestCases{
		{
			"set default sample duration when env var is missing",
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:          testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName:     testNamespace,
				config.NetworkNameEnvVarName:                   testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:              testNamespace,
				config.DesiredMaxLatencyMillisecondsEnvVarName: fmt.Sprintf("%d", testDesiredMaxLatency.Milliseconds()),
			},
			config.Config{
				CheckupParameters: config.CheckupParameters{
					SampleDurationSeconds:                config.DefaultSampleDuration,
					NetworkAttachmentDefinitionName:      testNetAttachDefName,
					NetworkAttachmentDefinitionNamespace: testNamespace,
					DesiredMaxLatencyMilliseconds:        testDesiredMaxLatency,
				},
				ResultsConfigMapName:      testResultConfigMapName,
				ResultsConfigMapNamespace: testNamespace,
			},
		},
		{
			"set default desired max latency when env var is missing",
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      testResultConfigMapName,
				config.ResultsConfigMapNamespaceEnvVarName: testNamespace,
				config.NetworkNameEnvVarName:               testNetAttachDefName,
				config.NetworkNamespaceEnvVarName:          testNamespace,
				config.SampleDurationSecondsEnvVarName:     fmt.Sprintf("%.0f", testSampleDuration.Seconds()),
			},
			config.Config{
				CheckupParameters: config.CheckupParameters{
					DesiredMaxLatencyMilliseconds:        config.DefaultDesiredMaxLatency,
					NetworkAttachmentDefinitionName:      testNetAttachDefName,
					NetworkAttachmentDefinitionNamespace: testNamespace,
					SampleDurationSeconds:                testSampleDuration,
				},
				ResultsConfigMapName:      testResultConfigMapName,
				ResultsConfigMapNamespace: testNamespace,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			testConfig, err := config.New(testCase.env)
			assert.NoError(t, err)
			assert.Equal(t, testConfig, testCase.expectedConfig)
		})
	}
}

type configCreateFallingTestCases struct {
	description   string
	expectedError error
	env           map[string]string
}

func TestCreateConfigFromEnvShouldFailWhen(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			"env is nil",
			config.ErrInvalidEnv,
			nil,
		},
		{
			"env is empty",
			config.ErrInvalidEnv,
			map[string]string{},
		},
		{
			"results ConfigMap name env var is missing",
			config.ErrResultsConfigMapNameMissing,
			map[string]string{
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"results ConfigMap name env var value is not valid",
			config.ErrInvalidResultsConfigMapName,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"results ConfigMap namespace env var is missing",
			config.ErrResultsConfigMapNamespaceMissing,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName: "results",
				config.NetworkNameEnvVarName:          "blue-net",
				config.NetworkNamespaceEnvVarName:     "default",
			},
		},
		{
			"results ConfigMap namespace env var value is not valid",
			config.ErrInvalidResultsConfigMapNamespace,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"network name env var is missing",
			config.ErrNetworkNameMissing,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"network name env var value is not valid",
			config.ErrInvalidNetworkName,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "",
				config.NetworkNamespaceEnvVarName:          "default",
			},
		},
		{
			"network namespace env var is missing",
			config.ErrNetworkNamespaceMissing,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
			},
		},
		{
			"network namespace env var value is not valid",
			config.ErrInvalidNetworkNamespace,
			map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.New(testCase.env)
			assert.Equal(t, err, testCase.expectedError)
		})
	}
}

func TestCreateConfigShouldFailWhenIntegerEnvVarsAreInvalid(t *testing.T) {
	testCases := []configCreateFallingTestCases{
		{
			description: "sample duration not valid integer",
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:      "results",
				config.ResultsConfigMapNamespaceEnvVarName: "default",
				config.NetworkNameEnvVarName:               "blue-net",
				config.NetworkNamespaceEnvVarName:          "default",
				config.SampleDurationSecondsEnvVarName:     "3rr0r",
			},
		},
		{
			description: "desired max latency is invalid",
			env: map[string]string{
				config.ResultsConfigMapNameEnvVarName:          "results",
				config.ResultsConfigMapNamespaceEnvVarName:     "default",
				config.NetworkNameEnvVarName:                   "blue-net",
				config.NetworkNamespaceEnvVarName:              "default",
				config.DesiredMaxLatencyMillisecondsEnvVarName: "39213801928309128309",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			_, err := config.New(testCase.env)
			assert.Error(t, err)
		})
	}
}
