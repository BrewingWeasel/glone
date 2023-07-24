package glone

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

type DirStructure []FileValues

type FileValues struct {
	Path        string `json:"path"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func getGitDir(link string) (DirStructure, error) {
	var result DirStructure

	resp, err := http.Get(link)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}
	return result, nil
}

func DealWithDir(link string, outputPrefix string) error {
	var wg sync.WaitGroup

	result, err := getGitDir(link)
	if err != nil {
		return err
	}

	for _, v := range result {
		wg.Add(1)
		if v.Type == "dir" {
			go func(val FileValues) {
				defer wg.Done()
				if err := os.MkdirAll(path.Join(outputPrefix, val.Path), os.ModePerm); err != nil {
					panic(err)
				}
				err := DealWithDir(val.URL, outputPrefix)
				if err != nil {
					panic(err)
				}

			}(v)
		} else {
			go func(val FileValues) {
				defer wg.Done()
				err := DownloadIndividualFile(val.DownloadURL, path.Join(outputPrefix, val.Path))
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

func DownloadIndividualFile(url string, fileName string) error {
	err := os.MkdirAll(path.Dir(fileName), os.ModePerm)
	if err != nil {
		return err
	}
	fmt.Println("\033[34mDownloading", url, "\033[m")
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
	fmt.Println("\033[32mDownloaded", fileName, "\033[m")
	return err
}

func DownloadSpecificFiles(url string, filePaths []string, output string) error {
	result, err := getGitDir(GetContsFile(url, ""))
	if err != nil {
		return err
	}
	branchName := strings.Split(result[0].URL, "?ref=")[1]
	for _, f := range filePaths {
		err := DownloadIndividualFile("https://raw.githubusercontent.com"+strings.TrimPrefix(url, "https://github.com")+"/"+branchName+"/"+f, path.Join(output, f))
		if err != nil {
			return err
		}
	}
	return nil
}
