package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"nickandperla.net/genetic_sort"

	"github.com/BurntSushi/toml"
)

var toolConfigPath *string = flag.String("config", "./config.toml", "The config file for genetic_sort tools to use. Defaults to './config.toml'")

var popId *uint = flag.Uint("popid", 1, "The id of the population you'd like to progress one generation")

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

	if persist, err := genetic_sort.NewPersistence(toolConfig.Persistence); err != nil {
		log.Fatalf("Failed to create or initialize Persistence: %v", err)
	} else {
		defer persist.Shutdown()
		if pop1, err := persist.LoadShallow(*popId); err != nil {
			log.Fatalf("Unable to load population from DB: %v", err)
		} else {
			if pop1.GetAliveCount() == 0 {
				log.Fatalf("Population [%d] has no living units", pop1.ID)
			}
			pop1.ProcessGeneration()

			fmt.Printf("Population [%d] unit count: %d\n", pop1.ID, pop1.GetAliveCount())
		}
	}

}
