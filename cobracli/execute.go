// Copyright 2016 Palantir Technologies. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cobracli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Execute executes the provided root command configured with the provided parameters. Returns an integer that should be
// used as the exit code for the application. Typical usage is "os.Exit(cobracli.Execute(...))" in a main function.
func Execute(rootCmd *cobra.Command, params ...Param) int {
	executor := &executor{}
	for _, p := range params {
		if p == nil {
			continue
		}
		p.apply(executor)
	}

	for _, configureCmd := range executor.configureCmds {
		configureCmd(rootCmd)
	}

	err := rootCmd.Execute()
	if err == nil {
		// command ran successfully: return 0
		return 0
	}

	// print error if error-printing function is defined
	if executor.errorHandler != nil {
		executor.errorHandler(rootCmd, err)
	}

	// extract custom exit code if exit code extractor is defined
	if executor.exitCodeExtractor != nil {
		return executor.exitCodeExtractor(err)
	}

	return 1
}

type executor struct {
	configureCmds     []func(*cobra.Command)
	errorHandler      func(*cobra.Command, error)
	exitCodeExtractor func(error) int
}

type Param interface {
	apply(*executor)
}

type paramFunc func(*executor)

func (f paramFunc) apply(e *executor) {
	f(e)
}

// ExitCodeExtractorParam sets the exit code extractor function for the executor. If the root command returns an error,
// the error is provided to the function and the code returned by the extractor is used as the exit code.
func ExitCodeExtractorParam(extractor func(error) int) Param {
	return paramFunc(func(executor *executor) {
		executor.exitCodeExtractor = extractor
	})
}

// ErrorHandlerParam sets the error handler for the command. If the root command returns and error, it is provided to
// the error handler for processing.
func ErrorHandlerParam(handler func(*cobra.Command, error)) Param {
	return paramFunc(func(executor *executor) {
		executor.errorHandler = handler
	})
}

// ErrorPrinterWithDebugHandler returns an error handler that prints the provided error as "Error: <error.Error()>"
// unless "error.Error()" is empty, in which case nothing is printed. If the provided boolean variable pointer is
// non-nil and the value is true, then the error output is provided to the specified error transform function before
// being printed.
func ErrorPrinterWithDebugHandler(debugVar *bool, debugErrTransform func(error) string) func(*cobra.Command, error) {
	return func(command *cobra.Command, err error) {
		errStr := err.Error()
		if errStr == "" {
			return
		}
		if debugVar != nil && *debugVar && debugErrTransform != nil {
			errStr = debugErrTransform(err)
		}
		command.Println("Error:", errStr)
	}
}

// ConfigureCmdParam adds the provided configuration function to the executor. All of the configuration functions on the
// executor are applied to the root command before it is executed.
func ConfigureCmdParam(configureCmd func(*cobra.Command)) Param {
	return paramFunc(func(executor *executor) {
		executor.configureCmds = append(executor.configureCmds, configureCmd)
	})
}

// RemoveHelpCommandConfigurer removes the "help" subcommand from the provided command.
func RemoveHelpCommandConfigurer(command *cobra.Command) {
	// set help command to be empty hidden command to effectively remove the built-in help command. Needs to be done in
	// this manner rather than by removing it because otherwise the default "Execute" logic will re-add the default
	// help command implementation.
	command.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})
}

// SilenceErrorsConfigurer configures the provided command to silence the default behavior of printing errors and
// printing command usage on errors.
func SilenceErrorsConfigurer(command *cobra.Command) {
	command.SilenceErrors = true
	command.SilenceUsage = true
}

// FlagErrorsUsageErrorConfigurer configures the provided command such that, when it encounters an error processing a
// flag, the returned error includes the usage string for the command.
func FlagErrorsUsageErrorConfigurer(command *cobra.Command) {
	command.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fmt.Errorf("%s\n%s", err.Error(), strings.TrimSuffix(c.UsageString(), "\n"))
	})
}
