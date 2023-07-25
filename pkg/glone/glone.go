package glone

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
		fmt.Println("Error unmarshalling JSON. This could be due to hitting a rate limit.")
		os.Exit(1)
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

FILES:
	for _, v := range result {

		if slices.Contains(config.Avoid, v.Path) {
			continue
		}

		byteStr := []byte(v.Path)
		for _, filter := range config.Filter {
			if matches, _ := regexp.Match(filter, byteStr); matches {
				if !config.Quiet {
					fmt.Println("Skipping", v.Path)
				}
				continue FILES
			}
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

func GetContsFile(normalLink string, path string) string {
	return strings.Replace(normalLink, "https://github.com", "https://api.github.com/repos", 1) + "/contents/" + path
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
