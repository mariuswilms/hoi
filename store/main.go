// Copyright 2016 Atelier Disko. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Provides in-memory store with a naive persisting option.
package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/atelierdisko/hoi/project"
)

func New(file string) *Store {
	return &Store{
		file: file,
		data: make(map[string]Entity),
	}
}

type Entity struct {
	Project project.Config
	Meta    project.Meta
}

type Store struct {
	sync.RWMutex
	// The database file handle, where store is persisting to.
	file string
	// Holds no pointers as it then would be possible to modify data outside lock.
	data           map[string]Entity
	autoSaverQuits chan struct{}
}

// Loads database file contents into memory. Does not hold an open handle
// on the file.
func (s *Store) Load() error {
	if _, err := os.Stat(s.file); os.IsNotExist(err) {
		return nil // nothing to do
	}
	log.Printf("loading db file: %s", s.file)

	f, err := os.Open(s.file)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "#", 2)

		if len(fields) != 2 {
			return errors.New("db file corrupt or in unrecognized format")
		}
		entity := &Entity{}
		err := json.Unmarshal([]byte(fields[1]), entity)

		if err != nil {
			return err
		}
		s.data[fields[0]] = *entity
	}
	return scanner.Err()
}

// Persists data into database file.
func (s Store) Store() error {
	// Swap contents at the very end, when we are sure that
	// everything else worked.
	var buf []byte
	b := bytes.NewBuffer(buf)

	for id, entity := range s.data {
		c, err := json.Marshal(entity)
		if err != nil {
			return err
		}
		b.WriteString(fmt.Sprintf("%s#%s\n", id, string(c)))
	}
	f, err := os.OpenFile(s.file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, b); err != nil {
		return err
	}
	return f.Sync()
}

// Wil periodically update the database file.
func (s *Store) InstallAutoStore() {
	ticker := time.NewTicker(1 * time.Minute)
	s.autoSaverQuits = make(chan struct{})
	log.Print("will periodically sync to database file")

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Print("auto storing")
				s.Lock()
				if err := s.Store(); err != nil {
					log.Print("failed to auto store")
				}
				s.Unlock()
			case <-s.autoSaverQuits:
				log.Print("uninstalled auto store")
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Store) Close() error {
	if s.autoSaverQuits != nil {
		close(s.autoSaverQuits)
	}
	s.Lock()
	defer s.Unlock()

	return s.Store()
}

func (s Store) Has(id string) bool {
	_, hasKey := s.data[id]
	return hasKey
}

func (s Store) Read(id string) (project.Config, error) {
	entity, hasKey := s.data[id]
	if !hasKey {
		return entity.Project, fmt.Errorf("failed to read from store: no id %s", id)
	}
	return entity.Project, nil
}

func (s Store) ReadAll() []Entity {
	all := make([]Entity, len(s.data))

	for _, entity := range s.data {
		all = append(all, entity)
	}
	return all
}

func (s *Store) Write(id string, pCfg project.Config) error {
	s.data[id] = Entity{
		Project: pCfg,
		Meta:    project.Meta{Status: project.StatusUnknown},
	}
	return nil
}

func (s *Store) Delete(id string) error {
	if _, hasKey := s.data[id]; !hasKey {
		return fmt.Errorf("failed to delete from store: no id %s", id)
	}
	delete(s.data, id)
	return nil
}

func (s Store) ReadStatus(id string) (project.MetaStatus, error) {
	if _, hasKey := s.data[id]; !hasKey {
		return project.StatusUnknown, fmt.Errorf("failed to read status: no id %s", id)
	}
	return s.data[id].Meta.Status, nil
}

func (s *Store) WriteStatus(id string, status project.MetaStatus) error {
	if _, hasKey := s.data[id]; !hasKey {
		return fmt.Errorf("failed to write status %s: no id %s", status, id)
	}
	entity := s.data[id]
	entity.Meta.Status = status
	s.data[id] = entity
	return nil
}
