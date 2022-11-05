package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

func main() {
	interactivePtr := flag.Bool("interactive", false, "Make results interactive with more details")
	githubTokenPtr := flag.String("githubToken", "", "Github API token")
	configPathPtr := flag.String("configpath", "./config.json", "Path to config file")
	limitProjectsPtr := flag.String("limitprojects", "", "Comma array of projects names to filter at query (ie: redis,rancher")

	flag.Parse()

	config := load_config(*configPathPtr)

	releases := get_releases(config.Projects, strings.Split(*limitProjectsPtr, ","), config.Providers, *githubTokenPtr)

	if *interactivePtr {

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "\U0001F336 {{ .Release_Element.Name | cyan }} ({{ .Release_Element.TagName | red }})",
			Inactive: "  {{ .Release_Element.Name | cyan }} ({{ .Release_Element.TagName | red }})",
			Selected: "\U0001F336 {{ .Release_Element.Name | red | cyan }}",
			Details: `
	--------- Release ----------
	{{ "Name:" | faint }}	{{ .Release_Element.Name }}
	{{ "URL:" | faint }}	{{ .Release_Element.URL }}
	{{ "TagName:" | faint }}	{{ .Release_Element.TagName }}`,
		}

		prompt := promptui.Select{
			Label:     "Select a Release",
			Items:     releases.Github_Releases,
			Templates: templates,
		}

		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		fmt.Printf("You choose %q\n", result)

	}

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

func get_releases(projects Projects, limitedprojects []string, providers []Provider, githubTokenPtr string) Filtered_Projects {

	var github_projects []Release_Element
	for _, project := range projects.Github {
		is := "github"
		for _, provider := range providers {
			if provider.Name == is {
				for _, limited := range limitedprojects {
					if limited == project.Project || limited == "" {
						fmt.Println("repo:", project.Owner, "/", project.Project)
						fmt.Println(provider)

						url := fmt.Sprint(provider.Url, "/repos/%s/%s/releases")
						releases := get_tags_github(project, url, githubTokenPtr)

						for _, release := range releases {
							if !release.Prerelease || (release.Prerelease && project.AllowPrerelease) {
								if strings.Contains(release.TagName, project.FilterMust) {
									fmt.Println(release.TagName)
									github_projects = append(github_projects, releases...)
								}

							}
						}

					}
				}
			}
		}

	}

	var dockerhub_projects []Result
	for _, project := range projects.Dockerhub {
		is := "dockerhub"
		for _, provider := range providers {
			if provider.Name == is {
				for _, limited := range limitedprojects {
					if limited == project.Project || limited == "" {
						fmt.Println("repo:", project.Project)
						fmt.Println(provider)

						url := fmt.Sprint(provider.Url, "/v2/repositories/library/%s/tags")
						releases := get_tags_dockerhub(project, url, githubTokenPtr)

						for _, release := range releases.Results {

							if strings.Contains(release.Name, project.FilterMust) {
								fmt.Println(release.Name)
								dockerhub_projects = append(dockerhub_projects, releases.Results...)
							}

						}
					}
				}
			}
		}
	}

	filtered_projects := Filtered_Projects{
		Github_Releases: github_projects,
		DockerHub_Tags:  dockerhub_projects,
	}
	return filtered_projects
}
