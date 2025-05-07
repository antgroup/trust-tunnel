// Copyright The TrustTunnel Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"encoding/json"
	"fmt"
	"os"
	"trust-tunnel/pkg/common/logutil"
	"trust-tunnel/pkg/trust-tunnel-agent/auth"
	"trust-tunnel/pkg/trust-tunnel-agent/backend"
	"trust-tunnel/pkg/trust-tunnel-agent/session"
	"trust-tunnel/pkg/trust-tunnel-agent/sidecar"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Option defines the options for the trust-tunnel-agent server.
type Option struct {
	Host            string                  `toml:"host"`
	Port            string                  `toml:"port"`
	SessionConfig   backend.SessionConfig   `toml:"session_config"`
	LogConfig       logutil.Config          `toml:"log_config"`
	TLSConfig       TLSConfig               `toml:"tls_config"`
	NTLSConfig      NTLSConfig              `toml:"ntls_config"`
	AuthConfig      auth.Config             `toml:"auth_config"`
	ContainerConfig session.ContainerConfig `toml:"container_config"`
	SidecarConfig   sidecar.Config          `toml:"sidecar_config"`
}

var (
	Version    string
	configPath string
)

// NewCommand creates and returns a new cobra command object.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust-tunnel-agent",
		Short: "trust-tunnel-agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			var options Option
			if err := loadConfigFromToml(&options); err != nil {
				return fmt.Errorf("failed to load config from toml: %w", err)
			}
			if err := runServer(&options); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "config.toml", "path to the config file")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display the current version of trust-tunnel-agent",
		Long:  "Display the current version of trust-tunnel-agent",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}
	cmd.AddCommand(versionCmd)

	return cmd
}

// loadConfigFromToml loads the configuration from the given TOML file.
func loadConfigFromToml(config *Option) error {
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", configPath, err)
	}

	return nil
}

// logGlobalConfig logs the global configuration.
func logGlobalConfig(opt *Option) {
	logrus.Info("trust-tunnel-agent start...")

	b, _ := json.Marshal(opt)
	logrus.Infof("config: %#v", string(b))
}
