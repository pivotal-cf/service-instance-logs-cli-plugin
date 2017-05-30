/*
 * Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
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
 */
package cli

import "fmt"
import "code.cloudfoundry.org/cli/cf/flags"

const (
	RecentUsage            = "Dump recent logs instead of tailing"
	SkipSslValidationUsage = "Skip verification of the logs endpoint. Not recommended!"
)

func ParseFlags(args []string) (bool, bool, []string, error) {
	const (
		recentFlagName        = "recent"
		sslValidationFlagName = "skip-ssl-validation"
	)

	fc := flags.New()
	//New flag methods take arguments: name, short_name and usage of the string flag
	fc.NewBoolFlag(recentFlagName, recentFlagName, RecentUsage)
	fc.NewBoolFlag(sslValidationFlagName, sslValidationFlagName, SkipSslValidationUsage)
	err := fc.Parse(args...)
	if err != nil {
		return false, false, nil, fmt.Errorf("Error parsing arguments: %s", err)
	}
	skipSslValidation := fc.Bool(sslValidationFlagName)
	recent := fc.Bool(recentFlagName)
	return recent, skipSslValidation, fc.Args(), nil
}
