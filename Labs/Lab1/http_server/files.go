package main

import (
	"Lab1/http"
	"errors"
	"fmt"
	"io"
	builtInHttp "net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Vne vad vi ska ha för Dir här, chat la in denna
var baseDir = "./public"

// används för content-type fältet, dessa är dom som är tillåtna enl. labbinstruktion
func checkContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".html":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".css":
		return "text/css"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

// line 33-68 gör lite olika säkerhetskontrollen innan själva filen faktiskt läses in
// Denna lösningen är inte optimal för stora filer, vet inte hur avancerat vi ska göra det
func myReadFunction(reqPath string) ([]byte, string, error) {
	fmt.Println(reqPath)
	clean := filepath.Clean(reqPath)
	full := filepath.Join(baseDir, clean)

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return nil, "", err
	}

	// Förhindra path traversal: absFull måste ligga under absBase
	if !strings.HasPrefix(absFull, absBase) {
		return nil, "", os.ErrPermission
	}

	// Hantera kataloger (t.ex. / -> index.html)
	fi, err := os.Stat(absFull)
	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}
	if fi.IsDir() {
		return nil, "", os.ErrInvalid
	}

	// Kontrollera filänderlsen (om inte tillåten returnerar vi tom ctype och låter handleren svara 400)
	ext := strings.ToLower(filepath.Ext(absFull))
	contentType := checkContentType(ext)

	//här läser vi själva filen
	data, err := os.ReadFile(absFull)
	if err != nil {
		return nil, "", err
	}

	return data, contentType, nil
}

// hanterar GET-förfrågningar
func FileGetHandler(r *builtInHttp.Request, rb *http.ResponseBuilder) error {
	p := r.URL.Path

	data, contentType, err := myReadFunction(p)
	if err != nil {
		status := http.InternalServerError

		if os.IsNotExist(err) {
			status = http.NotFound
		} else if os.IsPermission(err) {
			status = http.BadRequest
		} else if errors.Is(err, os.ErrInvalid) {
			status = http.BadRequest
		}

		return &http.HTTPError{Status: status}
	}

	if contentType == "" {
		return &http.HTTPError{Status: http.BadRequest}
	}

	rb.Bytes(data).Header("Content-Type", contentType)

	return nil
}

// FilePostHandler hanterar filuppladdningar.
// eller en raw POST där URL-path innehåller målfilens namn. Endast tillåtna filändelser sparas (enl. labbinstruktion)
func FilePostHandler(r *builtInHttp.Request, rb *http.ResponseBuilder) error {
	// contentType := r.Header.Get("Content-Type")
	var filename string
	var reader io.Reader

	// Raw body: use URL path as filename
	filename = filepath.Base(r.URL.Path)
	if filename == "" || filename == "/" {
		return &http.HTTPError{Status: http.BadRequest}
	}
	reader = r.Body
	defer r.Body.Close()

	ext := strings.ToLower(filepath.Ext(filename))
	if checkContentType(ext) == "" {
		return &http.HTTPError{Status: http.BadRequest}
	}

	// Säker join och abs-path kontroll
	dest := filepath.Join(baseDir, filepath.Clean("/"+filename))
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}
	if !strings.HasPrefix(absDest, absBase) {
		return &http.HTTPError{Status: http.BadRequest}
	}

	// Skriv till temporär fil och byt sedan namn (atomiskt)
	id := uuid.New()
	tmp, err := os.CreateTemp(baseDir, "upload-"+id.String())
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, reader); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return &http.HTTPError{Status: http.InternalServerError}
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return &http.HTTPError{Status: http.InternalServerError}
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return &http.HTTPError{Status: http.InternalServerError}
	}
	if err := os.Rename(tmpName, absDest); err != nil {
		os.Remove(tmpName)
		return &http.HTTPError{Status: http.InternalServerError}
	}

	rb.Status(http.Created)
	return nil
}
