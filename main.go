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

func main() {
	githubTokenPtr := flag.String("githubToken", "", "Github API token")
	configPathPtr := flag.String("configpath", "./config.json", "Path to config file")
	limitProjectsPtr := flag.String("limitprojects", "", "Comma array of projects names to filter at query (ie: redis,rancher")

	flag.Parse()

	config := load_config(*configPathPtr)

	get_releases(config.Projects, strings.Split(*limitProjectsPtr, ","), config.Providers, *githubTokenPtr)

	os.Exit(0)
}

func error_manager(error error, code uint16) {
	fmt.Println(error)

	switch code {
	case 1:
		fmt.Println("Error while loading config")
		panic(error)
	case 2:
		fmt.Println("Error while loading projects")
		panic(error)
	}

}

func get_tags_github(project Github, githubUrl string, githubToken string) Github_Releases {
	url := fmt.Sprintf(githubUrl, project.Owner, project.Project)

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

	var releases Github_Releases
	json.Unmarshal([]byte(body), &releases)

	return releases
}

func get_tags_dockerhub(project Dockerhub, providerurl string, providerToken string) DockerHub_Tags {

	url := fmt.Sprintf(providerurl, project.Project)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		error_manager(err, 2)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		error_manager(err, 2)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		error_manager(err, 2)
	}

	var releases DockerHub_Tags
	json.Unmarshal([]byte(body), &releases)

	return releases
}

func get_releases(projects Projects, limitedprojects []string, providers []Provider, githubTokenPtr string) {

	for _, project := range projects.Github {
		is := "github"
		for _, provider := range providers {
			if provider.Name == is {
				for _, limited := range limitedprojects {
					if limited == project.Project || limited == "" {
						fmt.Println("repo:", project.Owner, "/", project.Project)
						fmt.Println(providers)

						url := fmt.Sprint(provider.Url, "/repos/%s/%s/releases")
						releases := get_tags_github(project, url, githubTokenPtr)

						for _, release := range releases {
							switch release.Prerelease {
							case false:
								if strings.Contains(release.TagName, project.FilterMust) {
									fmt.Println(release.TagName)
								}

							case true && project.AllowPrerelease:
								if strings.Contains(release.TagName, project.FilterMust) {
									fmt.Println(release.TagName)
								}
							}
						}

					}
				}
			}
		}

	}

	for _, project := range projects.Dockerhub {

		for _, limited := range limitedprojects {
			if limited == project.Project || limited == "" {
				fmt.Println("repo:", project.Project)
				fmt.Println(providers)

				releases := get_tags_dockerhub(project, "https://hub.docker.com/v2/repositories/library/%s/tags", githubTokenPtr)

				for _, release := range releases.Results {

					if strings.Contains(release.Name, project.FilterMust) {
						fmt.Println(release.Name)
					}

				}
			}
		}
	}
}
