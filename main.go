package main

import (
	"encoding/json"
	"flag"
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
	githubTokenPtr := flag.String("githubToken", "", "Github API token")
	configPathPtr := flag.String("configpath", "./config.json", "Path to config file")
	limitProjectsPtr := flag.String("limitprojects", "", "Comma array of projects to filter at query")

	// This declares `numb` and `fork` flags, using a
	// similar approach to the `word` flag.
	// numbPtr := flag.Int("numb", 42, "an int")
	// boolPtr := flag.Bool("fork", false, "a bool")

	// It's also possible to declare an option that uses an
	// existing var declared elsewhere in the program.
	// Note that we need to pass in a pointer to the flag
	// declaration function.
	// var svar string
	// flag.StringVar(&svar, "svar", "bar", "a string var")

	// Once all flags are declared, call `flag.Parse()`
	// to execute the command-line parsing.
	flag.Parse()

	config := load_config(*configPathPtr)

	get_releases(config.Projects, strings.Split(*limitProjectsPtr, ","), config.Providers, *githubTokenPtr)

}

func error_manager(error error, code uint16) {
	fmt.Println(error)

	switch code {
	case 1:
		os.Exit(1)
	case 2:
		fmt.Println("Error while loading projects")
		os.Exit(1)
	}

}

func load_config(path string) Config {

	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		error_manager(err, 1)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		error_manager(err, 1)
	}
	fmt.Println("Successfully Opened", path)

	var config Config
	json.Unmarshal([]byte(byteValue), &config)

	// fmt.Println(config)
	return config
}

func get_releases_github(project Project, githubUrl string, githubToken string) Releases {
	url := fmt.Sprintf(githubUrl, project.GitOwner, project.GitProject)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		error_manager(err, 2)
	}

	req.Header.Add("Accept", "application/vnd.github+json")

	// if len(*githubTokenPtr) > 0 {
	// 	req.Header.Add("Authorization", "Bearer ")
	// }

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		error_manager(err, 2)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		error_manager(err, 2)
	}

	var releases Releases
	json.Unmarshal([]byte(body), &releases)

	return releases
}

func get_releases(projects []Project, limitedprojects []string, providers []Provider, githubTokenPtr string) {

	for _, project := range projects {

		for _, limited := range limitedprojects {
			if limited == project.GitProject || limited == "" {
				switch project.ProviderType {
				case "github":
					fmt.Println("repo:", project.GitOwner, "/", project.GitProject)
					fmt.Println(providers)

					releases := get_releases_github(project, "https://api.github.com/repos/%s/%s/releases", githubTokenPtr)

					for _, release := range releases {
						switch release.Prerelease {
						case false:
							if strings.Contains(release.TagName, project.GitReleaseFilterMust) {
								fmt.Println(release.TagName)
							}

						case true && project.GitAllowPrerelease:
							if strings.Contains(release.TagName, project.GitReleaseFilterMust) {
								fmt.Println(release.TagName)
							}
						}
					}

					// case "docker.io":

				}
			}
		}
	}

}
