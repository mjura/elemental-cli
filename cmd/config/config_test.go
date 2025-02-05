/*
Copyright © 2021 SUSE LLC

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

package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/mount-utils"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI config test suite")
}

var _ = Describe("Config", func() {
	Describe("Build config", Label("config", "build"), func() {
		It("values empty if config path not valid", Label("path", "values"), func() {
			cfg, err := ReadConfigBuild("/none/")
			Expect(err).To(BeNil())
			Expect(viper.GetString("label")).To(Equal(""))
			Expect(cfg.Label).To(Equal(""))
		})
		It("values filled if config path valid", Label("path", "values"), func() {
			cfg, err := ReadConfigBuild("config/")
			Expect(err).To(BeNil())
			Expect(viper.GetString("label")).To(Equal("COS_LIVE"))
			Expect(cfg.Label).To(Equal("COS_LIVE"))
		})
		It("overrides values with env values", Label("env", "values"), func() {
			_ = os.Setenv("ELEMENTAL_LABEL", "environment")
			cfg, err := ReadConfigBuild("config/")
			Expect(err).To(BeNil())
			source := viper.GetString("label")
			// check that the final value comes from the env var
			Expect(source).To(Equal("environment"))
			Expect(cfg.Label).To(Equal("environment"))
		})

	})
	Describe("Run config", Label("config", "run"), func() {
		var mounter mount.Interface

		BeforeEach(func() {
			mounter = &mount.FakeMounter{}
		})

		It("values empty if config does not exist", Label("path", "values"), func() {
			cfg, err := ReadConfigRun("/none/", mounter)
			Expect(err).To(BeNil())
			source := viper.GetString("file")
			Expect(source).To(Equal(""))
			Expect(cfg.Source).To(Equal(""))
		})
		It("values empty if config value is empty", Label("path", "values"), func() {
			cfg, err := ReadConfigRun("", mounter)
			Expect(err).To(BeNil())
			source := viper.GetString("file")
			Expect(source).To(Equal(""))
			Expect(cfg.Source).To(Equal(""))
		})
		It("overrides values with config files", Label("path", "values"), func() {
			cfg, err := ReadConfigRun("config/", mounter)
			Expect(err).To(BeNil())
			source := viper.GetString("target")
			// check that the final value comes from the extra file
			Expect(source).To(Equal("extra"))
			Expect(cfg.Target).To(Equal("extra"))
		})
		It("overrides values with env values", Label("path", "values"), func() {
			_ = os.Setenv("ELEMENTAL_TARGET", "environment")
			cfg, err := ReadConfigRun("config/", mounter)
			Expect(err).To(BeNil())
			source := viper.GetString("target")
			// check that the final value comes from the env var
			Expect(source).To(Equal("environment"))
			Expect(cfg.Target).To(Equal("environment"))
		})
		It("sets log level debug based on debug flag", Label("flag", "values"), func() {
			// Default value
			cfg, err := ReadConfigRun("config/", mounter)
			Expect(err).To(BeNil())
			debug := viper.GetBool("debug")
			Expect(cfg.Logger.GetLevel()).ToNot(Equal(logrus.DebugLevel))
			Expect(debug).To(BeFalse())

			// Set it via viper, like the flag
			viper.Set("debug", true)
			cfg, err = ReadConfigRun("config/", mounter)
			Expect(err).To(BeNil())
			debug = viper.GetBool("debug")
			Expect(debug).To(BeTrue())
			Expect(cfg.Logger.GetLevel()).To(Equal(logrus.DebugLevel))
		})
	})
})
