package github

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/brewingweasel/glone/pkg/gitservice"
)
import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type GithubFuncs struct{}

func (_ GithubFuncs) GetResponse(link string) ([]byte, error) {
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

func (_ GithubFuncs) HandleBranchUrl(url string, branch string) string {
	if branch == "" {
		return url
	} else {
		return strings.Split(url, "?ref=")[0] + "?ref=" + branch
	}
}

func (_ GithubFuncs) GetContsFile(normalLink string, path string) string {
	return strings.Replace(normalLink, "https://github.com", "https://api.github.com/repos", 1) + "/contents/" + path
}

func (_ GithubFuncs) GetBranch(url string) (string, error) {
	type RepoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	var result RepoInfo

	apiUrl := strings.Replace(url, "https://github.com", "https://api.github.com/repos", 1)
	response, err := GithubFuncs{}.GetResponse(apiUrl)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return "", err
	}
	return result.DefaultBranch, nil
}

func (_ GithubFuncs) GetDownloadFromPath(url string, branch string, path string) string {
	return "https://raw.githubusercontent.com" + strings.TrimPrefix(url, "https://github.com") + "/" + branch + "/" + path
}

func (_ GithubFuncs) GetGitDir(link string, _ string, _ string) (gitservice.DirStructure, error) {
	var result gitservice.DirStructure

	body, err := GithubFuncs{}.GetResponse(link)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatal("Error unmarshalling JSON. This could be due to hitting a rate limit. Create a github token and assign $GLONE_GITHUB_TOKEN to it in order to get more api calls")
	}
	return result, nil
}

func (_ GithubFuncs) GetTarball(url string, branch string) (string, error) {
	var downloadUrl string

	if branch == "" {
		branch, err := GithubFuncs{}.GetBranch(url)
		if err != nil {
			return downloadUrl, err
		}
		// TODO: adapt
		downloadUrl = url + "/tarball/" + branch
	} else {
		downloadUrl = url + "/tarball/" + branch
	}
	return downloadUrl, nil

}
