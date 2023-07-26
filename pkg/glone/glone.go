package glone

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/exp/slices"
)

type DirStructure []FileValues

type FileValues struct {
	Path        string `json:"path"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

type Config struct {
	Filter       []string
	Quiet        bool
	OutputPrefix string
	Avoid        []string
	Branch       string
	Path         string
}

func getResponse(link string) ([]byte, error) {
	var body []byte
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return body, err
	}

	token := os.Getenv("GLONE_GITHUB_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}
	return body, err

}

func GetGitDir(link string) (DirStructure, error) {
	var result DirStructure

	body, err := getResponse(link)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatal("Error unmarshalling JSON. This could be due to hitting a rate limit. Create a github token and assign $GLONE_GITHUB_TOKEN to it in order to get more api calls")
	}
	return result, nil
}

func DealWithDir(link string, getResult func(string) (DirStructure, error), config Config) error {
	var wg sync.WaitGroup

	handleUrl := func(url string) string {
		if config.Branch == "" {
			return url
		} else {
			return strings.Split(url, "?ref=")[0] + "?ref=" + config.Branch
		}
	}

	result, err := getResult(handleUrl(link))
	if err != nil {
		return err
	}

	for _, v := range result {

		if skipFile(v.Path, config) {
			continue
		}

		wg.Add(1)
		if v.Type == "dir" {
			go func(val FileValues) {
				defer wg.Done()
				if err := os.MkdirAll(path.Join(config.OutputPrefix, val.Path), os.ModePerm); err != nil {
					panic(err)
				}
				err := DealWithDir(handleUrl(val.URL), getResult, config)
				if err != nil {
					panic(err)
				}

			}(v)
		} else {
			go func(val FileValues) {
				defer wg.Done()
				err := DownloadIndividualFile(val.DownloadURL, path.Join(config.OutputPrefix, val.Path), config.Quiet)
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
	if strings.HasPrefix(origLink, "https://github.com") {
		return strings.Trim(origLink, "/")
	} else {
		return "https://github.com/" + strings.Trim(origLink, "/")
	}
}

func GetContsFile(normalLink string, path string) string {
	return strings.Replace(normalLink, "https://github.com", "https://api.github.com/repos", 1) + "/contents/" + path
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
	result, err := GetGitDir(GetContsFile(url, ""))
	if err != nil {
		return err
	}
	branchName := strings.Split(result[0].URL, "?ref=")[1]
	for _, f := range filePaths {
		err := DownloadIndividualFile("https://raw.githubusercontent.com"+strings.TrimPrefix(url, "https://github.com")+"/"+branchName+"/"+f, path.Join(config.OutputPrefix, f), config.Quiet)
		if err != nil {
			return err
		}
	}
	return nil
}

func getBranch(url string) (string, error) {
	type RepoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	var result RepoInfo

	apiUrl := strings.Replace(url, "https://github.com", "https://api.github.com/repos", 1)
	response, err := getResponse(apiUrl)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", err
	}
	return result.DefaultBranch, nil

}

func isPathShared(path1 string, path2 string) (bool, string) {
	first := strings.Split(path1, string(os.PathSeparator))[0]
	second := strings.Split(path2, string(os.PathSeparator))[0]
	return first == second, first
}

func DownloadTarball(url string, config Config) error {

	var downloadUrl string

	if config.Branch == "" {
		branch, err := getBranch(url)
		if err != nil {
			return err
		}
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
