package glone

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/brewingweasel/glone/pkg/github"
	"github.com/brewingweasel/glone/pkg/gitlab"
	"github.com/brewingweasel/glone/pkg/gitservice"
	"golang.org/x/exp/slices"
)

type gitHostingService interface {
	GetResponse(link string) ([]byte, error)
	HandleBranchUrl(url string, branch string) string
	GetContsFile(normalLink string, path string) string
	GetBranch(url string) (string, error)
	GetDownloadFromPath(url string, branch string, path string) string
	GetGitDir(link string, origLink string, branch string) (gitservice.DirStructure, error)
}

type Config struct {
	Filter       []string
	Quiet        bool
	ExcludePath  bool
	OutputPrefix string
	Avoid        []string
	Branch       string
	Path         string
	FileUrl      string
	Tar          bool
	GitHoster    gitHostingService
}

func RunGlone(config Config, specificFiles []string) {

	if strings.Contains(config.FileUrl, "github.com") {
		config.GitHoster = github.GithubFuncs{}
	} else {
		config.GitHoster = gitlab.GitlabFuncs{}
	}

	if len(specificFiles) != 0 {
		err := DownloadSpecificFiles(config.FileUrl, specificFiles, config)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	} else if config.Tar {
		err := DownloadTarball(config.FileUrl, config)
		if err != nil {
			log.Fatal(err)
		} else {
			os.Exit(0)
		}
	}

	err := DealWithDir(config.GitHoster.GetContsFile(config.FileUrl, config.Path), config)
	if err != nil {
		panic(err)
	}

}

func parsePath(path string, config Config) string {
	if config.ExcludePath {
		return strings.TrimPrefix(path, config.Path)
	} else {
		return path
	}
}

func DealWithDir(link string, config Config) error {
	var wg sync.WaitGroup

	handleUrl := func(url string) string {
		if config.Branch == "" {
			return url
		} else {
			return config.GitHoster.HandleBranchUrl(url, config.Branch)
		}
	}

	result, err := config.GitHoster.GetGitDir(handleUrl(link), config.FileUrl, config.Branch)
	if err != nil {
		return err
	}

	for _, v := range result {

		if skipFile(v.Path, config) {
			continue
		}

		wg.Add(1)
		if v.Type == "dir" || v.Type == "tree" {
			go func(val gitservice.FileValues) {
				defer wg.Done()
				if err := os.MkdirAll(path.Join(config.OutputPrefix, parsePath(val.Path, config)), os.ModePerm); err != nil {
					panic(err)
				}
				err := DealWithDir(handleUrl(val.URL), config)
				if err != nil {
					panic(err)
				}

			}(v)
		} else {
			go func(val gitservice.FileValues) {
				defer wg.Done()
				err := DownloadIndividualFile(val.DownloadURL, path.Join(config.OutputPrefix, parsePath(val.Path, config)), config.Quiet)
				if err != nil {
					panic(err)
				}
			}(v)
		}
	}
	wg.Wait()
	return nil
}

func NormalizeLink(origLink string) string {
	if strings.HasPrefix(origLink, "https://github.com") || !strings.Contains(origLink, "github.com") {
		return strings.Trim(origLink, "/")
	} else {
		return "https://github.com/" + strings.Trim(origLink, "/")
	}
}

func skipFile(path string, config Config) bool {
	if slices.Contains(config.Avoid, path) {
		return true
	}

	byteStr := []byte(path)
	for _, filter := range config.Filter {
		if matches, _ := regexp.Match(filter, byteStr); matches {
			if !config.Quiet {
				fmt.Println("\033[31mRegex matched, skipping: ", path, "\033[m")
			}
			return true
		}
	}
	return false
}

func DownloadIndividualFile(url string, fileName string, quiet bool) error {
	err := os.MkdirAll(path.Dir(fileName), os.ModePerm)
	if err != nil {
		return err
	}
	if !quiet {
		fmt.Println("\033[34mDownloading", url, "\033[m")
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if !quiet {
		fmt.Println("\033[32mDownloaded", fileName, "\033[m")
	}
	return err
}

func DownloadSpecificFiles(url string, filePaths []string, config Config) error {
	var branchName string
	var err error
	if config.Branch == "" {
		branchName, err = config.GitHoster.GetBranch(url)
		if err != nil {
			return err
		}
	} else {
		branchName = config.Branch
	}
	for _, f := range filePaths {
		err := DownloadIndividualFile(config.GitHoster.GetDownloadFromPath(url, branchName, f), path.Join(config.OutputPrefix, f), config.Quiet)
		if err != nil {
			return err
		}
	}
	return nil
}

func isPathShared(path1 string, path2 string) (bool, string) {
	first := strings.Split(path1, string(os.PathSeparator))[0]
	second := strings.Split(path2, string(os.PathSeparator))[0]
	return first == second, first
}

func DownloadTarball(url string, config Config) error {

	var downloadUrl string

	if config.Branch == "" {
		branch, err := config.GitHoster.GetBranch(url)
		if err != nil {
			return err
		}
		// TODO: adapt
		downloadUrl = url + "/tarball/" + branch
	} else {
		downloadUrl = url + "/tarball/" + config.Branch
	}

	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !config.Quiet {
		fmt.Println("\033[32mDownloaded tarball from ", downloadUrl, "\033[m")
	}

	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	tr := tar.NewReader(uncompressedStream)

	// Ignore the "pax_global_header" file
	tr.Next()
	// By default the directory name is not the name of the repository
	dirName, err := tr.Next()
	err = os.Mkdir(config.OutputPrefix, os.ModePerm)
	if err != nil {
		return err
	}

	replaceDirName := func(orig string) string {
		return strings.Replace(orig, dirName.Name, config.OutputPrefix+"/", 1)
	}

	remainingPath := config.Path

	for {
		hdr, err := tr.Next()

		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return err
		}

		name := replaceDirName(hdr.Name)

		if remainingPath != "" {
			matched, dir := isPathShared(strings.TrimPrefix(hdr.Name, dirName.Name), remainingPath)
			if matched {
				remainingPath = strings.TrimPrefix(remainingPath, dir+"/")
			} else {
				continue
			}
		}

		if skipFile(strings.TrimPrefix(name, config.OutputPrefix+"/"), config) {
			if !config.Quiet {
				fmt.Println("skipped: ", name)
			}
			continue
		}

		if hdr.Typeflag == tar.TypeDir {
			os.Mkdir(replaceDirName(name), os.ModePerm)
		} else {
			out, err := os.Create(name)
			if err != nil {
				fmt.Println("darn")
				return err
			}
			defer out.Close()

			if !config.Quiet {
				fmt.Println("Extracted " + name)
			}
			if _, err = io.Copy(out, tr); err != nil {
				return err
			}
		}
	}
	return nil
}
