package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
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

func main() {
	fileUrl := "https://github.com/BrewingWeasel/radish"
	err := DealWithDir(GetContsFile(fileUrl))
	if err != nil {
		panic(err)
	}
	fmt.Println("lol gg downloaded")
}

func DealWithDir(link string) error {
	fmt.Println(link)
	var wg sync.WaitGroup
	var result DirStructure

	resp, err := http.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	for _, v := range result {
		wg.Add(1)
		if v.Type == "dir" {
			go func(val FileValues) {
				defer wg.Done()
				if err := os.MkdirAll(val.Path, os.ModePerm); err != nil {
					panic(err)
				}
				err := DealWithDir(val.URL)
				if err != nil {
					panic(err)
				}

			}(v)
		} else {
			go func(val FileValues) {
				defer wg.Done()
				err := DownloadIndividualFile(val.DownloadURL, val.Path)
				if err != nil {
					panic(err)
				}
			}(v)
		}
	}
	wg.Wait()
	return nil
}

func GetContsFile(normalLink string) string {
	return strings.Replace(normalLink, "https://github.com", "https://api.github.com/repos", 1) + "/contents"
}

func DownloadIndividualFile(url string, fileName string) error {
	fmt.Println("Visiting", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Creating file for", url)
	out, err := os.Create(fileName)
	if err != nil {
		fmt.Println("uh oh")
		return err
	}
	defer out.Close()

	fmt.Println("Writing to file", fileName)
	_, err = io.Copy(out, resp.Body)
	fmt.Println("Downloaded", url)
	return err
}
