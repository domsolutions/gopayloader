package payloader

import (
	"github.com/domsolutions/gopayloader/version"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gopayloader",
	Short: "Gopayloader v" + version.Version + " HTTP load testing cross-platform tool with optional jwt generation - supports HTTP/1.1, HTTP/2, HTTP/3",
	Long:  ``,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
