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
	"errors"

	v1 "github.com/rancher-sandbox/elemental/pkg/types/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addCosignFlags adds flags related to cosign
func addCosignFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("cosign", "", false, "Enable cosign verification (requires images with signatures)")
	cmd.Flags().StringP("cosign-key", "", "", "Sets the URL of the public key to be used by cosign validation")
}

// addPowerFlags adds flags related to power
func addPowerFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("reboot", "", false, "Reboot the system after install")
	cmd.Flags().BoolP("poweroff", "", false, "Shutdown the system after install")
}

// addSharedInstallUpgradeFlags add flags shared between install, upgrade and reset
func addSharedInstallUpgradeFlags(cmd *cobra.Command) {
	cmd.Flags().String("directory", "", "Use directory as source to install from")
	cmd.Flags().StringP("docker-image", "d", "", "Install a specified container image")
	cmd.Flags().BoolP("no-verify", "", false, "Disable mtree checksum verification (requires images manifests generated with mtree separately)")
	cmd.Flags().BoolP("strict", "", false, "Enable strict check of hooks (They need to exit with 0)")

	addCosignFlags(cmd)
	addPowerFlags(cmd)
}

func validateCosignFlags(log v1.Logger) error {
	if viper.GetString("cosign-key") != "" && !viper.GetBool("cosign") {
		return errors.New("'cosign-key' requires 'cosign' option to be enabled")
	}

	if viper.GetBool("cosign") && viper.GetString("cosign-key") == "" {
		log.Warnf("No 'cosign-key' option set, keyless cosign verification is experimental")
	}
	return nil
}

func validateSourceFlags(log v1.Logger) error {
	// docker-image and directory are mutually exclusive. Can't have your cake and eat it too.
	if viper.GetString("docker-image") != "" && viper.GetString("directory") != "" {
		msg := "flags docker-image and directory are mutually exclusive, please only set one of them"
		return errors.New(msg)
	}
	return nil
}

func validatePowerFlags(log v1.Logger) error {
	if viper.GetBool("reboot") && viper.GetBool("poweroff") {
		return errors.New("'reboot' and 'poweroff' are mutually exclusive options")
	}
	return nil
}

// validateUpgradeFlags is a helper call to check all the flags for the upgrade command
func validateInstallUpgradeFlags(log v1.Logger) error {
	if err := validateSourceFlags(log); err != nil {
		return err
	}
	if err := validateCosignFlags(log); err != nil {
		return err
	}
	if err := validatePowerFlags(log); err != nil {
		return err
	}
	return nil
}
