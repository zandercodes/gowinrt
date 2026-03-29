/**
 * File: cli.go
 * Project: cli
 * Created Date: 2026‑03‑28T19:24:31.3131+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑28T22:09:42.4242+01:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package cli

import (
	"errors"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/zandercodes/gowinrt/internal/gen"
	"github.com/zandercodes/gowinrt/internal/logger"
)

var cliDescShort = "gowinrt is a tool to generate Go bindings for Windows Runtime APIs."
var cliDescLong = `gowinrt is a tool to generate Go bindings for Windows Runtime APIs.
It allows you to easily call Windows APIs from Go code.`

var verbose *bool
var class *string
var filter *[]string
var inheritance *bool
var validateOnly *bool

var cliCmd = &cobra.Command{
	Short: cliDescShort,
	Long:  cliDescLong,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.NewLogger()
		if *verbose {
			log = logger.NewLoggerWithLevel(zerolog.DebugLevel)
		}
		log.Debug().Msg("Starting gowinrt...")
		if *class == "" || len(*class) == 0 {
			err := errors.New("class is required")
			log.Error().Err(err).Msg("No class specified")
			return err
		}

		cfg := &gen.Config{
			Debug:        *verbose,
			Class:        *class,
			Filters:      *filter,
			Inheritance:  *inheritance,
			ValidateOnly: *validateOnly,
		}
		return gen.Generate(cfg, log)
	},
}

func init() {
	verbose = cliCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	class = cliCmd.Flags().StringP("class", "c", "", "Specify the class to generate bindings for (e.g. Windows.Foundation.Uri)")
	filter = cliCmd.Flags().StringArrayP("method-filter", "f", []string{}, "Filter methods to generate bindings for (e.g. --method-filter=GetHashCode --method-filter=ToString)")
	inheritance = cliCmd.Flags().Bool("inheritance", false, "Include inherited interface methods")
	validateOnly = cliCmd.Flags().Bool("validate", false, "Only validate generated files match, do not write")
}

func Execute() error {
	return cliCmd.Execute()
}
