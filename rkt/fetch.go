// Copyright 2014 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"runtime"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/appc/spec/schema/types"
	"github.com/coreos/rkt/common/apps"
	"github.com/coreos/rkt/store"

	"github.com/coreos/rkt/Godeps/_workspace/src/github.com/spf13/cobra"
)

const (
	defaultOS   = runtime.GOOS
	defaultArch = runtime.GOARCH
)

var (
	cmdFetch = &cobra.Command{
		Use:   "fetch IMAGE_URL...",
		Short: "Fetch image(s) and store them in the local store",
		Run:   runWrapper(runFetch),
	}
)

func init() {
	cmdRkt.AddCommand(cmdFetch)
	// Disable interspersed flags to stop parsing after the first non flag
	// argument. All the subsequent parsing will be done by parseApps.
	// This is needed to correctly handle multiple IMAGE --signature=sigfile options
	cmdFetch.Flags().SetInterspersed(false)

	cmdFetch.Flags().Var((*appAsc)(&rktApps), "signature", "local signature file to use in validating the preceding image")
	cmdFetch.Flags().BoolVar(&flagStoreOnly, "store-only", false, "use only available images in the store (do not discover or download from remote URLs)")
	cmdFetch.Flags().BoolVar(&flagNoStore, "no-store", false, "fetch images ignoring the local store")
}

func runFetch(cmd *cobra.Command, args []string) (exit int) {
	if err := parseApps(&rktApps, args, cmd.Flags(), false); err != nil {
		stderr("fetch: unable to parse arguments: %v", err)
		return 1
	}

	if rktApps.Count() < 1 {
		stderr("fetch: must provide at least one image")
		return 1
	}

	if flagStoreOnly && flagNoStore {
		stderr("both --store-only and --no-store specified")
		return 1
	}

	s, err := store.NewStore(globalFlags.Dir)
	if err != nil {
		stderr("fetch: cannot open store: %v", err)
		return 1
	}
	ks := getKeystore()
	config, err := getConfig()
	if err != nil {
		stderr("fetch: cannot get configuration: %v", err)
		return 1
	}
	ft := &fetcher{
		imageActionData: imageActionData{
			s:             s,
			ks:            ks,
			headers:       config.AuthPerHost,
			dockerAuth:    config.DockerCredentialsPerRegistry,
			insecureFlags: globalFlags.InsecureFlags,
			debug:         globalFlags.Debug,
		},
		storeOnly: flagStoreOnly,
		noStore:   flagNoStore,
		withDeps:  true,
	}

	err = rktApps.Walk(func(app *apps.App) error {
		hash, err := ft.fetchImage(app.Image, app.Asc)
		if err != nil {
			return err
		}
		shortHash := types.ShortHash(hash)
		stdout(shortHash)
		return nil
	})
	if err != nil {
		stderr("%v", err)
		return 1
	}

	return
}
