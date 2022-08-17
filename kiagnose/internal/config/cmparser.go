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

package config

import (
	"errors"
	"strings"
	"time"

	"github.com/kiagnose/kiagnose/kiagnose/types"
)

var (
	ErrImageFieldIsMissing   = errors.New("image field is missing")
	ErrImageFieldIsIllegal   = errors.New("image field is illegal")
	ErrTimeoutFieldIsMissing = errors.New("timeout field is missing")
	ErrTimeoutFieldIsIllegal = errors.New("timeout field is illegal")
	ErrParamNameIsIllegal    = errors.New("param name is illegal")
)

type configMapParser struct {
	configMapRawData map[string]string
	Image            string
	Timeout          time.Duration
	Params           map[string]string
	ClusterRoleNames []string
	RoleNames        []string
}

func newConfigMapParser(configMapRawData map[string]string) *configMapParser {
	return &configMapParser{
		configMapRawData: configMapRawData,
		Params:           map[string]string{},
	}
}

func (cmp *configMapParser) Parse() error {
	if err := cmp.parseImageField(); err != nil {
		return err
	}

	if err := cmp.parseTimeoutField(); err != nil {
		return err
	}

	if err := cmp.parseParamsField(); err != nil {
		return err
	}

	cmp.parseClusterRoleNamesField()
	cmp.parseRoleNamesField()

	return nil
}

func (cmp *configMapParser) parseImageField() error {
	var exists bool

	cmp.Image, exists = cmp.configMapRawData[types.ImageKey]
	if !exists {
		return ErrImageFieldIsMissing
	}

	if cmp.Image == "" {
		return ErrImageFieldIsIllegal
	}

	return nil
}

func (cmp *configMapParser) parseTimeoutField() error {
	rawTimeout, exists := cmp.configMapRawData[types.TimeoutKey]
	if !exists {
		return ErrTimeoutFieldIsMissing
	}

	var err error
	cmp.Timeout, err = time.ParseDuration(rawTimeout)
	if err != nil {
		return ErrTimeoutFieldIsIllegal
	}

	return nil
}

func (cmp *configMapParser) parseParamsField() error {
	for k, v := range cmp.configMapRawData {
		if strings.HasPrefix(k, types.ParamNameKeyPrefix) {
			paramName := strings.TrimPrefix(k, types.ParamNameKeyPrefix)
			if paramName == "" {
				return ErrParamNameIsIllegal
			}

			cmp.Params[paramName] = v
		}
	}

	return nil
}

func (cmp *configMapParser) parseClusterRoleNamesField() {
	if rawClusterRoleNames := cmp.configMapRawData[types.ClusterRolesKey]; rawClusterRoleNames != "" {
		cmp.ClusterRoleNames = parseListSeparatedByNewlines(rawClusterRoleNames)
	}
}

func (cmp *configMapParser) parseRoleNamesField() {
	if rawRoleNames := cmp.configMapRawData[types.RolesKey]; rawRoleNames != "" {
		cmp.RoleNames = parseListSeparatedByNewlines(rawRoleNames)
	}
}

func parseListSeparatedByNewlines(rawString string) []string {
	trimmedString := strings.TrimSpace(rawString)
	return strings.Split(trimmedString, "\n")
}
