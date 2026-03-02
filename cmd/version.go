package cmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/latch/internal/output"
	"github.com/jackchuka/latch/internal/version"
	"github.com/spf13/cobra"
)

type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if output.Format(cmd) == output.FormatJSON {
			return output.JSON(os.Stdout, versionInfo{
				Version:   version.Version,
				Commit:    version.Commit,
				BuildDate: version.BuildDate,
			})
		}
		fmt.Printf("latch %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildDate)
		return nil
	},
}

func init() {
	output.AddFlag(versionCmd)
	rootCmd.AddCommand(versionCmd)
}
