package glone

import (
	"fmt"
	"os"
	"strings"

	"github.com/brewingweasel/glone/pkg/glone"
	"github.com/spf13/cobra"
)

var specificFiles []string
var matchesFiles []string
var filteredVals []string
var avoidedFiles []string
var quiet bool
var tar bool
var branch string
var excludePath bool
var buildMode bool
var rootCmd = &cobra.Command{
	Use:   "glone <url (if using github, you can only include the user and the repository)> <path in repository> <output path>",
	Short: "Glone is git clone without the git. It allows downloading specific directories as well as a list of files to not download.",
	Args:  cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}

		fileUrl := glone.NormalizeLink(args[0])
		var path string
		var outputDir string

		if buildMode {
			filteredVals = append(filteredVals, ".github", "LICENSE", "README", ".gitignore", "CONTRIBUTING", "CHANGELOG", "AUTHORS", ".editorconfig")
		}

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

		config := glone.Config{OutputPrefix: outputDir, Filter: filteredVals, Avoid: avoidedFiles, Quiet: quiet, Branch: branch, Path: path, ExcludePath: excludePath, FileUrl: fileUrl, Tar: tar, MatchesFiles: matchesFiles}
		glone.RunGlone(config, specificFiles)

	},
}

func init() {
	rootCmd.Flags().StringArrayVarP(&specificFiles, "file", "f", []string{}, "Download a specific file or files.")
	rootCmd.Flags().StringArrayVar(&matchesFiles, "file-matches", []string{}, "Download all files matching the regex pattern.")
	rootCmd.Flags().StringArrayVarP(&avoidedFiles, "avoid", "a", []string{}, "Ignore specific file(s) or directory(s) and do not download them")
	rootCmd.Flags().StringArrayVarP(&filteredVals, "filter", "F", []string{}, "Ignore and do not download any files that match these regex patterns")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not output any info while downloading")
	rootCmd.Flags().BoolVarP(&tar, "tar", "t", false, "Download the tar.gz file and do processing while extracting instead of visiting each page")
	rootCmd.Flags().BoolVar(&excludePath, "exclude-path", false, "exclude the specific path specified")
	rootCmd.Flags().BoolVarP(&buildMode, "build", "B", false, "automatically exclude common files not needed for building such as .github")
	rootCmd.Flags().StringVarP(&branch, "branch", "b", "", "Specific branch to download")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running glone: '%s'", err)
		os.Exit(1)
	}
}
