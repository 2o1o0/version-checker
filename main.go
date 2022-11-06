package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var interactive_mode bool

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}
	return "\n" + m.list.View()
}

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

	const defaultWidth = 20
	items := []list.Item{}
	for _, release := range releases.Github_Releases {
		items = append(items, item(release.TagName))
	}
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "What do you want for dinner?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}
	if _, err := tea.NewProgram(m).StartReturningModel(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
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

	if len(githubToken) > 0 {
		bearer_token := fmt.Sprint("Bearer ", githubToken)
		req.Header.Add("Authorization", bearer_token)
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
