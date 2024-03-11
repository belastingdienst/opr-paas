/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"
	"strings"

	"github.com/belastingdienst/opr-paas/internal/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var debug bool

// requireSubcommand returns an error if no sub command is provided
// This was copied from podman: `github.com/containers/podman/cmd/podman/validate/args.go
// Some small style changes to match skopeo were applied, but try to apply any
// bugfixes there first.
func requireSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		suggestions := cmd.SuggestionsFor(args[0])
		if len(suggestions) == 0 {
			return fmt.Errorf("unrecognized command `%[1]s %[2]s`\nTry '%[1]s --help' for more information", cmd.CommandPath(), args[0])
		}
		return fmt.Errorf("unrecognized command `%[1]s %[2]s`\n\nDid you mean this?\n\t%[3]s\n\nTry '%[1]s --help' for more information", cmd.CommandPath(), args[0], strings.Join(suggestions, "\n\t"))
	}
	return fmt.Errorf("missing command '%[1]s COMMAND'\nTry '%[1]s --help' for more information", cmd.CommandPath())
}

// createApp returns a cobra.Command, and the underlying globalOptions object, to be run or tested.
func createApp() *cobra.Command {

	rootCommand := &cobra.Command{
		Use:              "crypttool",
		Long:             "Various operations for paas secret encryption",
		RunE:             requireSubcommand,
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	rootCommand.Version = version.PAAS_VERSION

	rootCommand.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	rootCommand.AddCommand(
		decryptCmd(),
		encryptCmd(),
		reencryptCmd(),
		checkPaasCmd(),
		generateCmd(),
	)
	return rootCommand
}

func main() {
	rootCmd := createApp()
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
