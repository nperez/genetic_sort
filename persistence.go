package genetic_sort

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	sqlite "github.com/glebarez/sqlite"
	gorm "gorm.io/gorm"
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
