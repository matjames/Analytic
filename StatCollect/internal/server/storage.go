package server

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// Store abstracts storage backend for attachments and submission files
type Store interface {
	SaveSubmission(instanceID string, name string, data []byte) error
	SaveFile(instanceID string, fh *multipart.FileHeader) (string, error)
}

// FSStore implements Store using the local filesystem
type FSStore struct {
	BaseDir string
}

func (s *FSStore) ensureDir(instanceID string) (string, error) {
	d := filepath.Join(s.BaseDir, instanceID)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

func (s *FSStore) SaveSubmission(instanceID string, name string, data []byte) error {
	dir, err := s.ensureDir(instanceID)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

func (s *FSStore) SaveFile(instanceID string, fh *multipart.FileHeader) (string, error) {
	dir, err := s.ensureDir(instanceID)
	if err != nil {
		return "", err
	}
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	outPath := filepath.Join(dir, filepath.Base(fh.Filename))
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, f)
	if err != nil {
		return "", err
	}
	return filepath.Base(outPath), nil
}
