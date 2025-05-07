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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version of the client.
var Version string

type Option struct {
	SessionID        string
	Host             string
	Port             int
	Pod              string
	ContainerName    string
	ContainerID      string
	IP               string
	Type             string
	Interactive      bool
	Tty              bool
	LoginName        string
	LoginGroup       string
	UserName         string
	TLSVerify        bool
	NTLSVerify       bool
	TLSCert          string
	TLSKey           string
	TLSCa            string
	NTLSCa           string
	NTLSSignKey      string
	NTLSSignCert     string
	NTLSEncCert      string
	NTLSEncKey       string
	Cipher           string
	Cmd              []string
	Cpus             float64
	MemoryMB         int
	DisableCleanMode bool
}

// NewCommand creates a new cobra command for the trust-tunnel-client.
func NewCommand() *cobra.Command {
	options := &Option{}
	cmd := &cobra.Command{
		Use:   "trust-tunnel-client [OPTIONS] COMMAND [ARG...]",
		Short: "Run a command in a remote running container or physical host",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Cmd = args
			exitCode, err := runClient(options)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(-1)
			}
			os.Exit(exitCode)

			return nil
		},
	}

	// Create version sub command.
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display the current version of this CLI tool",
		Long:  "Display the current version of this CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			// Print the version of trust-tunnel-client.
			fmt.Println(Version)
		},
	}

	cmd.AddCommand(versionCmd)

	// Setup command flags and bind them to options.
	setupCmdFlags(cmd, options)

	return cmd
}

// setupCmdFlags sets up the command line flags.
func setupCmdFlags(cmd *cobra.Command, options *Option) {
	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.StringVarP(&options.SessionID, "session-id", "s", "", "Session ID to uniquely identify the session")
	flags.StringVarP(&options.Host, "host", "o", "", "Target agent server address")
	flags.IntVarP(&options.Port, "port", "p", 5006, "Target agent server port")
	flags.StringVarP(&options.Type, "type", "", "phys", "Connection type: 'phys' for physical or 'container' for container")
	flags.StringVarP(&options.Pod, "pod", "", "", "Name of the target pod")
	flags.StringVarP(&options.ContainerName, "cname", "", "", "Name of the target container")
	flags.StringVarP(&options.ContainerID, "cid", "", "", "ID of the target container")
	flags.StringVarP(&options.IP, "ip", "", "", "IP address of the target container")
	flags.BoolVarP(&options.Interactive, "interactive", "i", false, "Start an interactive session with Stdin enabled")
	flags.BoolVarP(&options.Tty, "tty", "t", false, "Allocate a TTY for the session")
	flags.StringVarP(&options.LoginName, "login-name", "l", "root", "Username for logging into the target host")
	flags.StringVarP(&options.LoginGroup, "login-group", "g", "", "User group for logging into the target host")
	flags.StringVarP(&options.UserName, "user-name", "u", "", "User issuing the command")
	flags.BoolVarP(&options.TLSVerify, "tls-verify", "", false, "Enable TLS and verify the server's certificate")
	flags.BoolVarP(&options.NTLSVerify, "ntls-verify", "", false, "Use ntls and verify remote")
	flags.StringVarP(&options.TLSCert, "tls-cert", "", "", "Path to the TLS certificate file for authentication")
	flags.StringVarP(&options.TLSKey, "tls-key", "", "", "Path to the TLS private key file for authentication")
	flags.StringVarP(&options.TLSCa, "tls-ca", "", "", "Path to the TLS CA certificate file to verify the server")
	flags.StringVarP(&options.NTLSCa, "ntls-ca", "", "", "Specify NTLS ca file")
	flags.StringVarP(&options.NTLSSignKey, "ntls-sign-key", "", "", "Specify NTLS sign key file")
	flags.StringVarP(&options.NTLSSignCert, "ntls-sign-cert", "", "", "Specify NTLS sign cert file")
	flags.StringVarP(&options.NTLSEncCert, "ntls-enc-cert", "", "", "Specify NTLS enc cert file")
	flags.StringVarP(&options.NTLSEncKey, "ntls-enc-key", "", "", "Specify NTLS enc key file")
	flags.StringVarP(&options.Cipher, "cipher", "", "", "Specify NTLS cipher")
	flags.Float64VarP(&options.Cpus, "cpus", "c", 1.0, "Amount of CPU resources for command execution (e.g., 0.5, 2.0)")
	flags.IntVarP(&options.MemoryMB, "memory", "m", 512, "Amount of memory (MB) for command execution")
	flags.BoolVarP(&options.DisableCleanMode, "disable-clean-mode", "d", false, "Disabling clean mode prevents the use of sidecars and nsenter")
}
