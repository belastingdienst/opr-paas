/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireSubcommand(t *testing.T) {
	// cmd := cobra.Command{}
	args := []string{}
	cmd := createApp()

	// missing subcommand
	expectedErrorMsg := fmt.Sprintf("missing command '%[1]s COMMAND'\nTry '%[1]s --help' for more information", cmd.CommandPath())
	out := requireSubcommand(cmd, args)
	assert.EqualErrorf(t, out, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, out)

	// simulate totally wrong command (i.e. no suggestions)
	args = []string{"unrecognizableCommand"}
	expectedErrorMsg = fmt.Sprintf("unrecognized command `%[1]s %[2]s`\nTry '%[1]s --help' for more information", cmd.CommandPath(), args[0])
	out = requireSubcommand(cmd, args)
	assert.EqualErrorf(t, out, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, out)

	// simulate typo
	args = []string{"decryp"}
	expectedErrorMsg = fmt.Sprintf("unrecognized command `%[1]s %[2]s`\n\nDid you mean this?\n\t%[3]s\n\nTry '%[1]s --help' for more information", cmd.CommandPath(), args[0], strings.Join([]string{"decrypt"}, "\n\t"))
	out = requireSubcommand(cmd, args)
	assert.EqualErrorf(t, out, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, out)
}
