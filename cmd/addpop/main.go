package main

import (
	"flag"
	"log"
	"os"

	gs "nickandperla.net/genetic_sort"

	"github.com/BurntSushi/toml"
)

/*
	Read config file (TOML)

	From unmarshaled config:
		Create population
		Connect/Initialize DB
		Persist new population

	return population id

*/

var populationConfigPath *string = flag.String("popconfig", "./pop.toml", "The population config to use when creating the population. Defaults to './pop.toml'")
var toolConfigPath *string = flag.String("config", "./config.toml", "The config file for genetic_sort tools to use. Defaults to './config.toml'")

func main() {
	flag.Parse()

	conffile, err := os.Open(*toolConfigPath)

	if err != nil {
		log.Fatalf("Unable to load genetic_sort config: %v", err)
	}

	confDecoder := toml.NewDecoder(conffile)
	var toolConfig gs.ToolConfig
	if _, err = confDecoder.Decode(&toolConfig); err != nil {
		log.Fatalf("Failed to unmarshal tool config: %v", err)
	}
	conffile.Close()

	popfile, err := os.Open(*populationConfigPath)

	if err != nil {
		log.Fatalf("Unable to load population config: %v", err)
	}

	popDecoder := toml.NewDecoder(popfile)
	var popConfig gs.PopulationConfig

	if _, err = popDecoder.Decode(&popConfig); err != nil {
		log.Fatalf("Failed to unmarshal population config: %v", err)
	}

	popfile.Close()

	log.Printf("ToolConfig: %+v\nPopConfig: %+v\n", toolConfig, popConfig)
}
