package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pavelpuchok/insightcourier/config"
)

type FileStorage struct {
	mu       *sync.Mutex
	filePath string
}

type fileStorageData struct {
	Sources map[string]time.Time `json:"sources"`
}

func NewFileStorage(cfg config.FileStorageConfig) (*FileStorage, error) {
	err := os.MkdirAll(path.Dir(cfg.FilePath), 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to create parent directories for storage file %s. %w", cfg.FilePath, err)
	}

	f := &FileStorage{
		mu:       new(sync.Mutex),
		filePath: cfg.FilePath,
	}

	_, err = f.readFile()
	if os.IsNotExist(err) {
		err = f.writeFile(fileStorageData{
			Sources: make(map[string]time.Time),
		})
	}

	return f, err
}

func (f *FileStorage) writeFile(d fileStorageData) error {
	data, err := json.Marshal(&d)
	if err != nil {
		return err
	}

	return os.WriteFile(f.filePath, data, 0644)
}

func (f *FileStorage) readFile() (*fileStorageData, error) {
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return nil, err
	}

	d := fileStorageData{
		Sources: make(map[string]time.Time),
	}

	err = json.Unmarshal(data, &d)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (f *FileStorage) GetSourceUpdateTime(source string) (*time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, err := f.readFile()
	if err != nil {
		return nil, fmt.Errorf("unable to read storage file %s. %w", f.filePath, err)
	}

	t, has := d.Sources[source]
	if !has {
		return nil, ErrSourceNotFound
	}

	return &t, nil
}

func (f *FileStorage) SetSourceUpdateTime(source string, t time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	d, err := f.readFile()
	if err != nil {
		return fmt.Errorf("unable to read storage file %s. %w", f.filePath, err)
	}
	d.Sources[source] = t
	err = f.writeFile(*d)
	if err != nil {
		return fmt.Errorf("unable to write storage file %s. %w", f.filePath, err)
	}

	return nil
}
