package main

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const publicDir = "./public"

var (
	allowedContentTypes = map[string]string{
		".html": "text/html",
		".txt":  "text/plain",
		".css":  "text/css",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
	}
	errUnsupportedContentType = errors.New("unsupported content type")
)

func contentTypeForPath(p string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(p))
	contentType, ok := allowedContentTypes[ext]
	return contentType, ok
}

func resolvePath(requestPath string) (string, error) {
	cleaned := path.Clean("/" + requestPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	fullPath := filepath.Join(publicDir, cleaned)

	absBase, err := filepath.Abs(publicDir)
	if err != nil {
		return "", err
	}

	absFull, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absBase, absFull)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(rel, "..") {
		return "", os.ErrPermission
	}

	return absFull, nil
}

func readFile(requestPath string) ([]byte, error) {
	diskPath, err := resolvePath(requestPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(diskPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, os.ErrInvalid
	}

	data, err := os.ReadFile(diskPath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func saveUploadedFile(reader io.Reader, destination string) error {
	tmp, err := os.CreateTemp(publicDir, "upload-")
	if err != nil {
		return err
	}
	defer tmp.Close()

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	_, err = io.Copy(tmp, reader)
	if err != nil {
		return err
	}

	err = tmp.Sync()
	if err != nil {
		return err
	}

	err = tmp.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpName, destination)
	if err != nil {
		return err
	}

	return nil
}

func writeFile(requestPath string, reader io.Reader) error {
	filename := filepath.Base(requestPath)
	if filename == "" || filename == "." || filename == "/" || filename == string(os.PathSeparator) {
		return os.ErrInvalid
	}

	_, ok := contentTypeForPath(filename);
	if !ok {
		return errUnsupportedContentType
	}

	destination, err := resolvePath(filename)
	if err != nil {
		return err
	}

	return saveUploadedFile(reader, destination)
}
