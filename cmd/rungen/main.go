package main

import (
	"context"
	"flag"
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
			loaders := persist.GetUnitLoaders(pop1, toolConfig.BatchSize)
			evaluator := genetic_sort.NewEvaluator(pop1.PopulationConfig.EvaluatorConfig)
			selector := genetic_sort.NewSelector(pop1.PopulationConfig.SelectorConfig)
			engine := genetic_sort.NewGenerationEngine(loaders, evaluator, selector)

			ctx, _ := context.WithCancel(context.Background())
			go engine.Run(ctx)

			batch := make([]*genetic_sort.Unit, toolConfig.BatchSize)
			index := uint(0)
		FOR:
			for {
				select {
				case unit := <-engine.Output:
					if unit == nil {
						if batch[0] != nil {
							if genetic_sort.DEBUG {
								log.Printf("Persisting unit batch")
							}
							if err := persist.UpdateUnits(&batch); err != nil {
								log.Fatalf("Persisting batch of units failed: %v", err)
							}
						}
						break FOR
					}

					batch[index] = unit
					index++
					if index == toolConfig.BatchSize {
						if genetic_sort.DEBUG {
							log.Printf("Persisting unit batch")
						}
						if err := persist.UpdateUnits(&batch); err != nil {
							log.Fatalf("Persisting batch of units failed: %v", err)
						} else {
							for i := range batch {
								batch[i] = nil
							}
						}
						index = 0
					}
				case <-ctx.Done():
					break FOR
				}
			}

		}
	}

}
