package genetic_sort

type ToolConfig struct {
	BatchSize   uint               `toml:"batch_size"`
	Persistence *PersistenceConfig `toml:"persistence"`
}
