/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/rancher-sandbox/elemental/cmd/config"
	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/mount-utils"
)

// pullImage represents the pull-image command
var pullImage = &cobra.Command{
	Use:   "pull-image IMAGE DESTINATION",
	Short: "elemental pull-image",
	Args:  cobra.ExactArgs(2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return CheckRoot()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ReadConfigRun(viper.GetString("config-dir"), &mount.FakeMounter{})

		if err != nil {
			cfg.Logger.Errorf("Error reading config: %s\n", err)
		}

		image := args[0]
		destination, err := filepath.Abs(args[1])
		if err != nil {
			cfg.Logger.Errorf("Invalid path %s", destination)
			return err
		}

		// Set this after parsing of the flags, so it fails on parsing and prints usage properly
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true // Do not propagate errors down the line, we control them

		verify, _ := cmd.Flags().GetBool("verify")
		user, _ := cmd.Flags().GetString("auth-username")
		local, _ := cmd.Flags().GetBool("local")
		pass, _ := cmd.Flags().GetString("auth-password")
		authType, _ := cmd.Flags().GetString("auth-type")
		server, _ := cmd.Flags().GetString("auth-server-address")
		identity, _ := cmd.Flags().GetString("auth-identity-token")
		registryToken, _ := cmd.Flags().GetString("auth-registry-token")
		plugins, _ := cmd.Flags().GetStringArray("plugin")

		auth := &types.AuthConfig{
			Username:      user,
			Password:      pass,
			ServerAddress: server,
			Auth:          authType,
			IdentityToken: identity,
			RegistryToken: registryToken,
		}

		luet := v1.NewLuet(v1.WithLuetLogger(cfg.Logger), v1.WithLuetAuth(auth), v1.WithLuetPlugins(plugins...))
		luet.VerifyImageUnpack = verify
		err = luet.Unpack(destination, image, local)

		if err != nil {
			cfg.Logger.Error(err.Error())
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullImage)
	pullImage.Flags().String("auth-username", "", "Username to authenticate to registry/notary")
	pullImage.Flags().String("auth-password", "", "Password to authenticate to registry")
	pullImage.Flags().String("auth-type", "", "Auth type")
	pullImage.Flags().String("auth-server-address", "", "Authentication server address")
	pullImage.Flags().String("auth-identity-token", "", "Authentication identity token")
	pullImage.Flags().String("auth-registry-token", "", "Authentication registry token")
	pullImage.Flags().Bool("verify", false, "Verify signed images to notary before to pull")
	pullImage.Flags().Bool("local", false, "Use local image")
	pullImage.Flags().StringArray("plugin", []string{}, "A list of runtime plugins to load. Can be repeated to add more than one plugin")
}
