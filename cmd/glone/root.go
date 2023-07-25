package glone

import (
	"fmt"
	"os"
	"strings"

	"github.com/brewingweasel/glone/pkg/glone"
	"github.com/spf13/cobra"
)

var specificFiles []string
var filteredVals []string
var avoidedFiles []string
var quiet bool
var rootCmd = &cobra.Command{
	Use:   "glone url (path in repository) (output path)",
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

		config := glone.Config{OutputPrefix: outputDir, Filter: filteredVals, Avoid: avoidedFiles, Quiet: quiet}

		if len(specificFiles) != 0 {
			err := glone.DownloadSpecificFiles(fileUrl, specificFiles, config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running glone: '%s'", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		err := glone.DealWithDir(glone.GetContsFile(fileUrl, path), config)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.Flags().StringArrayVarP(&specificFiles, "file", "f", []string{}, "Download a specific file or files.")
	rootCmd.Flags().StringArrayVarP(&avoidedFiles, "avoid", "a", []string{}, "Ignore specific file(s) or directory(s) and do not download them")
	rootCmd.Flags().StringArrayVarP(&filteredVals, "filter", "F", []string{}, "Ignore and do not download any files that match these regex patterns")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not output any info while downloading")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running glone: '%s'", err)
		os.Exit(1)
	}
}
