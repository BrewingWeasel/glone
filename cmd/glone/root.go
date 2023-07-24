package glone

import (
	"fmt"
	"os"
	"strings"

	"github.com/brewingweasel/glone/pkg/glone"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "glone",
	Short: "Glone is an alternative to git clone that allows downloading specific directories",
	Args:  cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}

		fileUrl := args[0]
		var path string
		var outputDir string

		if len(args) > 1 {
			path = args[1]
		} else {
			path = ""
		}
		if len(args) == 3 {
			outputDir = args[2]
		} else {
			urlParts := strings.Split(strings.TrimSuffix(fileUrl, "/"), "/")
			outputDir = urlParts[len(urlParts)-1]
		}

		err := glone.DealWithDir(glone.GetContsFile(fileUrl, path), outputDir)
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
