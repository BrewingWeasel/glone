package gitlab

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/brewingweasel/glone/pkg/gitservice"
)
import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type GitlabFuncs struct{}

func (_ GitlabFuncs) GetResponse(link string) ([]byte, error) {
	var body []byte
	resp, err := http.Get(link)

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

func (_ GitlabFuncs) HandleBranchUrl(url string, branch string) string {
	if branch == "" {
		return url
	} else if strings.Contains(url, "?path=") {
		return fmt.Sprint(url, "&ref=", branch)
	} else {
		return fmt.Sprint(url, "?ref=", branch)
	}
}

func getApiUrl(normalLink string) string {
	urlParts := strings.Split(normalLink, "/")
	repo := fmt.Sprint(urlParts[len(urlParts)-2], "%2F", urlParts[len(urlParts)-1])
	mainUrl := strings.Join(urlParts[0:len(urlParts)-2], "/")
	return fmt.Sprint(mainUrl, "/api/v4/projects/", repo, "/repository")
}

func (_ GitlabFuncs) GetContsFile(normalLink string, path string) string {
	return fmt.Sprint(getApiUrl(normalLink), "/tree?path=", path)
}

func (_ GitlabFuncs) GetBranch(url string) (string, error) {
	type RepoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	var result RepoInfo

	apiUrl := getApiUrl(url)
	response, err := GitlabFuncs{}.GetResponse(apiUrl)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", err
	}
	return result.DefaultBranch, nil
}

func (_ GitlabFuncs) GetDownloadFromPath(url string, branch string, path string) string {
	return fmt.Sprint(url, "/-/raw/", branch, "/", path)
}

func (_ GitlabFuncs) GetGitDir(link string, origLink string, branch string) (gitservice.DirStructure, error) {
	var result gitservice.DirStructure

	body, err := GitlabFuncs{}.GetResponse(link)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatal("Error unmarshalling JSON")
	}

	for i := 0; i < len(result); i++ {
		file := &result[i]
		file.DownloadURL = GitlabFuncs{}.GetDownloadFromPath(origLink, branch, file.Path)
		file.URL = GitlabFuncs{}.GetContsFile(origLink, file.Path)
	}
	return result, nil
}

func (_ GitlabFuncs) GetTarball(url string, branch string) (string, error) {
	var downloadUrl string

	downloadUrl = fmt.Sprint(getApiUrl(url), "/archive?ref=", branch)
	return downloadUrl, nil
}
