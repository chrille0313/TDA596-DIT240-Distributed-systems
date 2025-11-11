package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	clean := filepath.Clean("/" + reqPath)
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
		return nil, "", err
	}
	if fi.IsDir() {
		index := filepath.Join(absFull, "index.html")
		if _, err := os.Stat(index); err == nil {
			absFull = index
		} else {
			return nil, "", os.ErrNotExist
		}
	}

	// Kontrollera filänderlsen (om inte tillåten returnerar vi tom ctype och låter handleren svara 400)
	ext := strings.ToLower(filepath.Ext(absFull))
	ctype := checkContentType(ext)

	//här läser vi själva filen
	data, err := os.ReadFile(absFull)
	if err != nil {
		return nil, "", err
	}

	return data, ctype, nil
}

// hanterar GET-förfrågningar
func FileGetHandler(r *http.Request, rb *ResponseBuilder) {
	p := r.URL.Path

	data, ctype, err := myReadFunction(p)
	if err != nil {
		if os.IsNotExist(err) {
			rb.Status(NotFound).Body("404 not found")
			return
		}
		if os.IsPermission(err) {
			rb.Status(BadRequest).Body("invalid path")
			return
		}
		rb.Status(InternalServerError).Body("internal error")
		return
	}

	if ctype == "" {
		rb.Status(BadRequest).Body("400 bad request")
		return
	}

	rb.Header("Content-Type", ctype)

	if strings.HasPrefix(ctype, "text/") || strings.Contains(ctype, "json") || strings.Contains(ctype, "xml") {
		rb.Body(string(data))
	} else {
		// Temporärt: konvertera även binärt till string (kan korrupta binära data).
		// Ett bättre tillvägagångssätt är att uppgradera Response/ResponseBuilder för []byte.
		rb.Body(string(data))
	}
	rb.Status(Ok)
}

// FilePostHandler hanterar filuppladdningar. Den accepterar multipart/form-data med fältet "file"
// eller en raw POST där URL-path innehåller målfilens namn. Endast tillåtna filändelser sparas (enl. labbinstruktion)
func FilePostHandler(r *http.Request, rb *ResponseBuilder) {
	// Maxstorlek för enkelhets skull (10 MB)
	const maxUploadSize = 10 << 20

	ct := r.Header.Get("Content-Type")
	var filename string
	var reader io.Reader

	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			rb.Status(BadRequest).Body("400 bad request")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			rb.Status(BadRequest).Body("400 bad request")
			return
		}
		defer file.Close()
		filename = filepath.Base(header.Filename)
		reader = file
	} else {
		// Raw body: use URL path as filename
		filename = filepath.Base(r.URL.Path)
		if filename == "" || filename == "/" {
			rb.Status(BadRequest).Body("400 bad request")
			return
		}
		reader = r.Body
		defer r.Body.Close()
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if checkContentType(ext) == "" {
		rb.Status(BadRequest).Body("400 bad request")
		return
	}

	// Säker join och abs-path kontroll
	dest := filepath.Join(baseDir, filepath.Clean("/"+filename))
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		rb.Status(InternalServerError).Body("internal error")
		return
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		rb.Status(InternalServerError).Body("internal error")
		return
	}
	if !strings.HasPrefix(absDest, absBase) {
		rb.Status(BadRequest).Body("400 bad request")
		return
	}

	// Skriv till temporär fil och byt sedan namn (atomiskt)
	tmp, err := os.CreateTemp(baseDir, "upload-*")
	if err != nil {
		rb.Status(InternalServerError).Body("internal error")
		return
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, reader); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		rb.Status(InternalServerError).Body("internal error")
		return
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		rb.Status(InternalServerError).Body("internal error")
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		rb.Status(InternalServerError).Body("internal error")
		return
	}
	if err := os.Rename(tmpName, absDest); err != nil {
		os.Remove(tmpName)
		rb.Status(InternalServerError).Body("internal error")
		return
	}

	rb.Status(Created).Body("201 created")
}
