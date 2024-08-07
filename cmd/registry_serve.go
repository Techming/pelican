/***************************************************************
 *
 * Copyright (C) 2024, Pelican Project, Morgridge Institute for Research
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you
 * may not use this file except in compliance with the License.  You may
 * obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***************************************************************/

package main

import (
	"github.com/spf13/cobra"

	"github.com/pelicanplatform/pelican/launchers"
	"github.com/pelicanplatform/pelican/server_structs"
)

func serveRegistry(cmd *cobra.Command, _ []string) error {
	_, cancel, err := launchers.LaunchModules(cmd.Context(), server_structs.RegistryType)
	if err != nil {
		cancel()
	}

	return err
}
