/**
 * File: cli.go
 * Project: cli
 * Created Date: 2026‑03‑28T19:24:31.3131+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑31T23:09:22.2222+02:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package cli

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/zandercodes/gowinrt/internal/emit"
	"github.com/zandercodes/gowinrt/internal/gen"
	"github.com/zandercodes/gowinrt/internal/logger"
	"github.com/zandercodes/gowinrt/internal/metadata"
	"github.com/zandercodes/gowinrt/internal/resolve"
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

		cfg := &resolve.Config{
			Debug:        *verbose,
			Class:        *class,
			Filters:      *filter,
			Inheritance:  *inheritance,
			ValidateOnly: *validateOnly,
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		// metadata → resolve → gen → emit pipeline
		store, err := metadata.NewStore(log)
		if err != nil {
			return err
		}

		td, err := store.TypeDefByName(cfg.Class)
		if err != nil {
			return fmt.Errorf("failed to get typedef for class %s: %w", cfg.Class, err)
		}

		resolver := resolve.NewResolver(store, cfg.MethodFilter(), cfg.Inheritance, log)
		dataFiles, err := resolver.ResolveType(td)
		if err != nil {
			return fmt.Errorf("failed to resolve type %s: %w", cfg.Class, err)
		}

		tmpl, err := gen.LoadTemplates()
		if err != nil {
			return fmt.Errorf("failed to load templates: %w", err)
		}

		emitter := emit.NewEmitter(tmpl, cfg.ValidateOnly)
		for _, f := range dataFiles {
			if err := emitter.EmitFile(f, td.Namespace.String()); err != nil {
				return fmt.Errorf("failed to emit file %s: %w", f.Filename, err)
			}
		}
		return nil
	},
}

func init() {
	verbose = cliCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	class = cliCmd.Flags().StringP("class", "c", "", "Specify the class to generate bindings for (e.g. Windows.Foundation.Uri)")
	filter = cliCmd.Flags().StringArrayP("method-filter", "f", []string{}, "Filter methods to generate bindings for (e.g. --method-filter=GetHashCode --method-filter=ToString)")
	inheritance = cliCmd.Flags().BoolP("inheritance", "i", false, "Include inherited interface methods")
	validateOnly = cliCmd.Flags().BoolP("validate", "V", false, "Only validate generated files match, do not write")
}

func Execute() error {
	return cliCmd.Execute()
}
