package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"nickandperla.net/genetic_sort"

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
	var toolConfig genetic_sort.ToolConfig
	if _, err = confDecoder.Decode(&toolConfig); err != nil {
		log.Fatalf("Failed to unmarshal tool config: %v", err)
	}
	conffile.Close()

	popfile, err := os.Open(*populationConfigPath)

	if err != nil {
		log.Fatalf("Unable to load population config: %v", err)
	}

	popDecoder := toml.NewDecoder(popfile)
	var popConfig genetic_sort.PopulationConfig

	if _, err = popDecoder.Decode(&popConfig); err != nil {
		log.Fatalf("Failed to unmarshal population config: %v", err)
	}

	popfile.Close()

	if persist, err := genetic_sort.NewPersistence(toolConfig.Persistence); err != nil {
		log.Fatalf("Failed to create or initialize Persistence: %v", err)
	} else {
		defer persist.Shutdown()
		if pop, err := persist.Create(&popConfig); err != nil {
			log.Fatalf("Failed to create population: %v", err)
		} else {
			pop.SynthesizeUnits()
			fmt.Printf("%d\n", pop.ID)
		}
	}
}
