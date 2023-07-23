package glone

import (
	"fmt"
	"os"

	"github.com/brewingweasel/glone/pkg/glone"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "glone",
	Short: "Glone is an alternative to git clone that allows downloading specific directories",
	Run: func(cmd *cobra.Command, args []string) {
		fileUrl := args[0]
		path := args[1]
		err := glone.DealWithDir(glone.GetContsFile(fileUrl, path))
		if err != nil {
			panic(err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running glone: '%s'", err)
		os.Exit(1)
	}
}
