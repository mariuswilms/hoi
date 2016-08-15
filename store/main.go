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

	"github.com/atelierdisko/hoi/project"
)

func New(file string) (*Store, error) {
	store := &Store{
		file: file,
		data: make(map[string]project.Config),
	}
	if err := store.Load(); err != nil {
		return store, err
	}
	log.Printf("in-memory store ready")
	return store, nil
}

type Store struct {
	// Global lock.
	sync.RWMutex
	// The database file handle, where store is persisting to.
	file string
	// Holds no pointers as it then would be possible to modify data outside lock.
	data map[string]project.Config
}

// Loads database file contents into memory. Does not hold an open handle
// on the file.
func (s *Store) Load() error {
	if _, err := os.Stat(s.file); os.IsNotExist(err) {
		return nil // nothing to do
	}
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
		cfg, _ := project.New()
		err := json.Unmarshal([]byte(fields[1]), cfg)

		if err != nil {
			return err
		}
		s.data[fields[0]] = *cfg
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Persists data into database file.
func (s *Store) Store() error {
	// Swap contents at the very end, when we are sure that
	// everything else worked.
	var buf []byte
	b := bytes.NewBuffer(buf)

	for id, cfg := range s.data {
		c, err := json.Marshal(cfg)
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
	return nil
}

func (s *Store) Close() error {
	s.Lock()
	defer s.Unlock()
	return s.Store()
}

func (s *Store) Has(id string) bool {
	_, hasKey := s.data[id]
	return hasKey
}

func (s *Store) Read(id string) (project.Config, error) {
	pCfg, hasKey := s.data[id]
	if !hasKey {
		return pCfg, fmt.Errorf("failed to read from store: no id %s", id)
	}
	return pCfg, nil
}

func (s *Store) ReadAll() map[string]project.Config {
	return s.data
}

func (s *Store) Write(id string, cfg project.Config) error {
	s.data[id] = cfg
	return nil
}

func (s *Store) Delete(id string) error {
	if _, hasKey := s.data[id]; !hasKey {
		return fmt.Errorf("failed to delete from store: no id %s", id)
	}
	delete(s.data, id)
	return nil
}

func (s *Store) Stats() string {
	s.RLock()
	defer s.RUnlock()
	return fmt.Sprintf("STATS STORE count:%d", len(s.data))
}
