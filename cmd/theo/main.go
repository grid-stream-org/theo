package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/grid-stream-org/go-commons/pkg/logger"
	"github.com/grid-stream-org/go-commons/pkg/sigctx"
	"github.com/grid-stream-org/theo/internal/config"
	"github.com/grid-stream-org/theo/internal/theo"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

func main() {
	log := logger.Default()
	exitCode := 0
	if err := run(); err != nil {
		exitCode = handleErrors(err, log)
	}
	log.Info("Done", "exitCode", exitCode)
	os.Exit(exitCode)
}

func run() (err error) {
	// Set up our signal handler
	ctx, cancel := sigctx.New(context.Background())
	defer cancel()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Initialize logger
	log, err := logger.New(cfg.Log, nil)
	if err != nil {
		return err
	}

	// Create Theo
	theo, err := theo.New(ctx, cfg, log)
	if err != nil {
		return err
	}

	// Check for timeout
	// Do not return the context cancellation error because we suppress them (to account for signals)
	if cfg.Theo.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Theo.Timeout)
		defer cancel()
	}

	// Run Theo
	err = theo.Run(ctx)

	// Check for timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Errorf("theo timed out after %s", cfg.Theo.Timeout)
	}

	return err
}

func handleErrors(err error, log *slog.Logger) int {
	if err == nil {
		return 0
	}
	var exitCode int
	errs := []error{}
	// Filter and process errors
	for _, mErr := range multierr.Errors(err) {
		var sigErr *sigctx.SignalError
		if errors.As(mErr, &sigErr) {
			exitCode = sigErr.SigNum()
		} else if !errors.Is(mErr, context.Canceled) {
			errs = append(errs, mErr)
		}
	}
	// Log non-signal errors
	if len(errs) > 0 {
		for _, err := range errs {
			log.Error("error occurred", "error", err, "stack", fmt.Sprintf("%+v", err))
		}
		if exitCode == 0 {
			exitCode = 255
		}
	}
	return exitCode
}
