package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"nickandperla.net/genetic_sort"

	"github.com/BurntSushi/toml"
)

var toolConfigPath = flag.String("config", "./config.toml", "The config file for genetic_sort tools to use")
var popId = flag.Uint("popid", 1, "The id of the population to prune")
var dryRun = flag.Bool("dry-run", false, "Preview what would be deleted without actually deleting")

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

	persist, err := genetic_sort.NewPersistence(toolConfig.Persistence)
	if err != nil {
		log.Fatalf("Failed to create or initialize Persistence: %v", err)
	}
	defer persist.Shutdown()

	pop, err := persist.LoadShallow(*popId)
	if err != nil {
		log.Fatalf("Unable to load population from DB: %v", err)
	}

	if *dryRun {
		log.Printf("DRY RUN: previewing prune for population %d", pop.ID)
	} else {
		log.Printf("Pruning dead lineages for population %d", pop.ID)
	}

	result, err := pop.Prune(*dryRun)
	if err != nil {
		log.Fatalf("Prune failed: %v", err)
	}

	fmt.Printf("Population %d prune %s:\n", pop.ID, map[bool]string{true: "(dry run)", false: "complete"}[*dryRun])
	fmt.Printf("  Total units:           %d\n", result.TotalUnits)
	fmt.Printf("  Alive units:           %d\n", result.AliveUnits)
	fmt.Printf("  Ancestor units kept:   %d\n", result.AncestorUnits)
	fmt.Printf("  Units deleted:         %d\n", result.DeletedUnits)
	fmt.Printf("  Instructions deleted:  %d\n", result.DeletedInstructions)
	fmt.Printf("  Mutations deleted:     %d\n", result.DeletedMutations)
	fmt.Printf("  Evaluations deleted:   %d\n", result.DeletedEvaluations)
	fmt.Printf("  Tombstones deleted:    %d\n", result.DeletedTombstones)
}
