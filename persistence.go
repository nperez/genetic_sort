package genetic_sort

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	sqlite "github.com/glebarez/sqlite"
	gorm "gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PersistenceConfig struct {
	Name          string
	Path          string
	SQLitePragmas []string
	SQLiteOptions []string
}

type Persistence struct {
	Config *PersistenceConfig
	DB     *gorm.DB
}

func NewPersistence(config *PersistenceConfig) (*Persistence, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Path) == 0 {
		return nil, fmt.Errorf("Path to database must be defined")
	}

	if len(config.Name) == 0 {
		return nil, fmt.Errorf("Name of database must be defined")
	}

	var pragmas strings.Builder
	pragma_count := len(config.SQLitePragmas) - 1
	for i, prag := range config.SQLitePragmas {
		pragmas.WriteString(fmt.Sprintf("_pragma=%s", prag))
		if i < pragma_count {
			pragmas.WriteRune('&')
		}
	}

	var options strings.Builder
	option_count := len(config.SQLiteOptions) - 1
	for i, opt := range config.SQLiteOptions {
		pragmas.WriteString(opt)
		if i < option_count {
			options.WriteRune('&')
		}
	}

	var path strings.Builder
	path.WriteString(filepath.Join(config.Path, config.Name))
	if pragmas.Len() > 0 {
		path.WriteRune('?')
		path.WriteString(pragmas.String())
		if options.Len() > 0 {
			path.WriteRune('&')
			path.WriteString(options.String())
		}
	} else if options.Len() > 0 {
		path.WriteRune('?')
		path.WriteString(options.String())
	}

	db, err := gorm.Open(sqlite.Open(path.String()), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	db = db.Session(&gorm.Session{PrepareStmt: true, CreateBatchSize: 1000})

	p := &Persistence{Config: config, DB: db}
	if err = p.initialize(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Persistence) initialize() error {
	if err := p.DB.AutoMigrate(
		&Population{},
		&Unit{},
		&Instruction{},
		&Mutation{},
		&Evaluation{},
		&Tombstone{},
	); err != nil {
		return err
	}

	return nil
}

func (p *Persistence) Shutdown() {
	if sqldb, err := p.DB.DB(); err != nil {
		log.Fatalf("Failed to retrieve raw DB: %v", err)
	} else {
		sqldb.Close()
	}
}

func (p *Persistence) Create(pop *Population) (uint, error) {
	if pop == nil {
		return 0, fmt.Errorf("Population cannot be nil")
	}

	if result := p.DB.Create(pop); result.Error != nil {
		return 0, fmt.Errorf("Failed to call gorm.Create(): %w", result.Error)
	}

	return pop.ID, nil

}

func (p *Persistence) LoadShallow(id uint) (*Population, error) {
	pop := &Population{}
	tx := p.DB.Find(pop, id)
	if tx.Error != nil {
		return nil, fmt.Errorf("Failed to find population [%d]: %w", id, tx.Error)
	}

	if pop.ID == 0 {
		return nil, fmt.Errorf("Population id [%d] not found", id)
	}
	return pop, nil
}

/*
	Figure out how to make sure all unit associations are updated. Right now, units are missing evaluations
*/

func (p *Persistence) UpdateUnits(units *[]*Unit) error {

	for _, unit := range *units {
		if len(unit.Evaluations) == 0 {
			log.Fatalf("Missing evaluation")
		}
	}
	tx := p.DB.Session(&gorm.Session{FullSaveAssociations: true})
	tx.Save(*units)
	if tx.Error != nil {
		return fmt.Errorf("Failed to save units: %w", tx.Error)
	}

	return nil
}

type UnitLoader func(id, total uint) <-chan []*Unit

func (p *Persistence) GetUnitLoaders(pop *Population, batchSize uint) []UnitLoader {
	cpus := runtime.NumCPU()
	ret := make([]UnitLoader, cpus)
	for i := 0; i < cpus; i++ {
		ret[i] = func(id, total uint) <-chan []*Unit {
			outpipe := make(chan []*Unit)

			go func() {
				if DEBUG {
					log.Printf("Starting unit loader %d/%d", id+1, total)
				}
				units := make([]*Unit, batchSize)
				p.DB.Model(&Unit{}).
					Preload("Instructions.Mutations").
					Preload(clause.Associations).
					Where("units.population_id = ?", pop.ID).
					Where("units.id % ? = ?", total, id).
					Where("units.alive = ?", Alive).
					FindInBatches(&units, int(batchSize), func(tx *gorm.DB, batchnum int) error {
						cunits := make([]*Unit, len(units))
						copy(cunits, units)
						outpipe <- cunits
						if DEBUG {
							log.Printf("Sent batch %d from loader %d", batchnum, id+1)
						}
						return nil
					})
				if DEBUG {
					log.Printf("Closing loader %d", id+1)
				}
				close(outpipe)
			}()

			return outpipe
		}
	}

	return ret
}
