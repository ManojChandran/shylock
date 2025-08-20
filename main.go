// Package main provides the entry point for the Shylock AWS cost estimation tool.
//
// Shylock is a CLI tool that estimates AWS costs based on resource configurations
// defined in JSON files. It supports multiple AWS services including EC2, ALB,
// RDS, Lambda, and S3, providing detailed cost breakdowns with assumptions and
// recommendations.
//
// Usage:
//
//	shylock estimate config.json [flags]
//	shylock validate config.json
//	shylock list
//	shylock version
//
// For detailed usage information, run: shylock --help
package main

import (
	"fmt"
	"os"

	"shylock/cmd"
	"shylock/internal/errors"
)

// main is the entry point for the Shylock application.
// It executes the CLI commands and handles error formatting and exit codes.
func main() {
	if err := cmd.Execute(); err != nil {
		// Format error for user display using structured error handling
		if estimationErr, ok := err.(*errors.EstimationError); ok {
			fmt.Fprint(os.Stderr, errors.FormatErrorForUser(estimationErr))
			os.Exit(errors.GetExitCode(estimationErr))
		} else {
			// Handle unexpected errors
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
