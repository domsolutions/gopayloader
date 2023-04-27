package payloader

import (
	"errors"
	"github.com/domsolutions/gopayloader/pkgs/payloader"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
)

var clearCacheCmd = &cobra.Command{
	Use:   "clear-cache",
	Short: "Delete all generated jwts",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		if payloader.JwtCacheDir == "" {
			return errors.New("cache directory couldn't be determined")
		}
		if err := os.RemoveAll(payloader.JwtCacheDir); err != nil {
			return err
		}
		pterm.Success.Println("Cache cleared")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearCacheCmd)
}
