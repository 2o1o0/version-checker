package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Providers []Provider `json:"providers"`
	Projects  []Project  `json:"projects"`
}

type Project struct {
	ProviderType         string `json:"provider_type"`
	GitOwner             string `json:"git_owner"`
	GitProject           string `json:"git_project"`
	GitReleaseFilterMust string `json:"git_release_filter_must"`
	GitAllowPrerelease   bool   `json:"git_allow_prerelease"`
}

type Provider struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

type Releases []Release_Element

type Release_Element struct {
	URL             string  `json:"url"`
	HTMLURL         string  `json:"html_url"`
	AssetsURL       string  `json:"assets_url"`
	UploadURL       string  `json:"upload_url"`
	TarballURL      string  `json:"tarball_url"`
	ZipballURL      string  `json:"zipball_url"`
	ID              int64   `json:"id"`
	NodeID          string  `json:"node_id"`
	TagName         string  `json:"tag_name"`
	TargetCommitish string  `json:"target_commitish"`
	Name            string  `json:"name"`
	Body            string  `json:"body"`
	Draft           bool    `json:"draft"`
	Prerelease      bool    `json:"prerelease"`
	CreatedAt       string  `json:"created_at"`
	PublishedAt     string  `json:"published_at"`
	Author          Author  `json:"author"`
	Assets          []Asset `json:"assets"`
}

type Asset struct {
	URL                string `json:"url"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ID                 int64  `json:"id"`
	NodeID             string `json:"node_id"`
	Name               string `json:"name"`
	Label              string `json:"label"`
	State              string `json:"state"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
	DownloadCount      int64  `json:"download_count"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	Uploader           Author `json:"uploader"`
}

type Author struct {
	Login             string `json:"login"`
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

func main() {

	config := load_config("config.json")

	get_releases(config.Projects, config.Providers)

}

func load_config(path string) Config {

	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened", path)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var config Config
	json.Unmarshal([]byte(byteValue), &config)

	// fmt.Println(config)
	return config
}

func get_releases(projects []Project, providers []Provider) {

	for _, project := range projects {

		switch project.ProviderType {
		case "github":
			fmt.Println("repo:", project.GitOwner, "/", project.GitProject)
			fmt.Println(providers)
			url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", project.GitOwner, project.GitProject)

			req, _ := http.NewRequest("GET", url, nil)

			req.Header.Add("Accept", "application/vnd.github+json")

			// if config.Providers["github"] {
			// 	req.Header.Add("Authorization", "Bearer ")
			// }

			res, _ := http.DefaultClient.Do(req)
			defer res.Body.Close()
			body, _ := ioutil.ReadAll(res.Body)

			var releases Releases
			json.Unmarshal([]byte(body), &releases)
			// fmt.Println(len(releases))

			var desiredReleases Releases
			for _, release := range releases {
				switch release.Prerelease {
				case false:
					if strings.Contains(release.TagName, project.GitReleaseFilterMust) {
						desiredReleases = append(desiredReleases, release)
					}

				case true && project.GitAllowPrerelease:
					if strings.Contains(release.TagName, project.GitReleaseFilterMust) {
						desiredReleases = append(desiredReleases, release)
					}
				}
			}

			for _, desiredRelease := range desiredReleases {
				fmt.Println(desiredRelease.TagName)
			}
		case "docker.io":

		}

	}

}
