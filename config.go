package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func create_config(path string) {
	config := Config{
		Providers: []Provider{
			{
				Name: "",
				Url:  "",
			},
		},
		Projects: Projects{
			Github: []Github{
				{
					Owner:           "organisation",
					Project:         "project",
					AllowPrerelease: false,
				},
			},
			Dockerhub: []Dockerhub{
				{
					Project: "project",
				},
			},
		},
	}

	file, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		error_manager(err, 1)
	}

	_ = ioutil.WriteFile("config.json", file, 0644)

	fmt.Println("Config file generated: ", path)
	os.Exit(0)
}

func load_config(path string) Config {

	jsonFile, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("File Does Not Exist: ", path)
			create_config(path)
		} else {
			error_manager(err, 1)
		}

	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		error_manager(err, 1)
	}
	fmt.Println("Successfully Opened", path)

	var config Config
	json.Unmarshal([]byte(byteValue), &config)

	return config
}
