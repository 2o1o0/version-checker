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

var interactive_mode bool

func main() {
	interactivePtr := flag.Bool("interactive", false, "Make results interactive with more details")
	githubTokenPtr := flag.String("githubToken", "", "Github API token")
	configPathPtr := flag.String("configpath", "./config.json", "Path to config file")
	limitProjectsPtr := flag.String("limitprojects", "", "Comma array of projects names to filter at query (ie: redis,rancher")

	flag.Parse()

	interactive_mode = *interactivePtr

	config := load_config(*configPathPtr)

	if interactive_mode {
		fmt.Println("Loading projects...")

	}

	releases := get_releases(config.Projects, strings.Split(*limitProjectsPtr, ","), config.Providers, *githubTokenPtr)

	if interactive_mode {

		prompt_provider := promptui.Select{
			Size:     10,
			Label:    "Select a Provider",
			Items:    config.Providers,
			HideHelp: true,
		}

		_, result_provider, err := prompt_provider.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		var templates *promptui.SelectTemplates
		var prompt promptui.Select

		if strings.Contains(result_provider, "github") {
			templates = &promptui.SelectTemplates{
			Label:    "{{ . }}?",
				Active:   "\u27A1\uFE0F {{ .Name | cyan }} ({{ .HTMLURL | red }})",
				Inactive: "  {{ .Name | cyan }}",
				Selected: "\u27A1\uFE0F {{ .Name | red | cyan }}",
			Details: `
	--------- Release ----------
				{{ "Name:" | faint }}	{{ .Name }}
				{{ "URL:" | faint }}	{{ .HTMLURL }}
				{{ "TagName:" | faint }}	{{ .TagName }}
				{{ "PreRelease:" | faint }}	{{ .Prerelease }}`,
		}

			prompt = promptui.Select{
				Size:      10,
			Label:     "Select a Release",
			Items:     releases.Github_Releases,
			Templates: templates,
				HideHelp:  true,
			}

		} else if strings.Contains(result_provider, "dockerhub") {
			templates = &promptui.SelectTemplates{
				Label:    "{{ . }}?",
				Active:   "\u27A1\uFE0F {{ .Name | cyan }}",
				Inactive: "  {{ .Name | cyan }}",
				Selected: "\u27A1\uFE0F {{ .Name | red | cyan }}",
				Details: `
				--------- Release ----------
				{{ "Name:" | faint }}	{{ .Name }}`,
			}

			prompt = promptui.Select{
				Size:      10,
				Label:     "Select a Release",
				Items:     releases.DockerHub_Tags,
				Templates: templates,
				HideHelp:  true,
			}

		} else {
			error_manager(err, 3)
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

	github_projects := Github_Releases{}
	for _, project := range projects.Github {
		is := "github"
		for _, provider := range providers {
			if provider.Name == is {
				for _, limited := range limitedprojects {
					if limited == project.Project || limited == "" {
						if !interactive_mode {
						fmt.Println("repo:", project.Owner, "/", project.Project)
						fmt.Println(provider)
						}

						url := fmt.Sprint(provider.Url, "/repos/%s/%s/releases")
						releases := get_tags_github(project, url, githubTokenPtr)

						for _, release := range releases {
							if !release.Prerelease || (release.Prerelease && project.AllowPrerelease) {
								if strings.Contains(release.TagName, project.FilterMust) {
									if !interactive_mode {
									fmt.Println(release.TagName)
									}
									github_projects = append(github_projects, release)
								}
								}

							}
						}

					}
				}
			}
		}

	dockerhub_projects := DockerHub_Tags{}
	for _, project := range projects.Dockerhub {
		is := "dockerhub"
		for _, provider := range providers {
			if provider.Name == is {
				for _, limited := range limitedprojects {
					if limited == project.Project || limited == "" {
						if !interactive_mode {
						fmt.Println("repo:", project.Project)
						fmt.Println(provider)
						}
						url := fmt.Sprint(provider.Url, "/v2/repositories/library/%s/tags")
						releases := get_tags_dockerhub(project, url, githubTokenPtr)

						for _, release := range releases.Results {

							if strings.Contains(release.Name, project.FilterMust) {
								if !interactive_mode {
								fmt.Println(release.Name)

								}
								dockerhub_projects.Results = append(dockerhub_projects.Results, release)
							}

						}
					}
				}
			}
		}
	}

	filtered_projects := Filtered_Projects{
		Github_Releases: github_projects,
		DockerHub_Tags:  dockerhub_projects.Results,
	}
	return filtered_projects
}
