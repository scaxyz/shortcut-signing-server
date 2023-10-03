package internal

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oklog/ulid/v2"
	"gopkg.in/yaml.v3"
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

type Server struct {
	options       serverOptions
	errorToString func(err error, short string) string // TODO better
	jobCounterMx  sync.Mutex
	jobCounter    int
}

type signingPayload struct {
	ShortcutName string
	Shortcut     string
}

var defaultServerOptions = []ServerOption{
	TempDir(os.TempDir()),
	MaxContentSize(10 * MB),
	MaxFilenameLength(255),
}

func NewServer(listen string, options ...ServerOption) *Server {
	server := &Server{
		errorToString: func(_ error, short string) string { return short },
	}

	for _, option := range defaultServerOptions {
		option(&server.options)
	}

	for _, option := range options {
		option(&server.options)
	}

	if server.options.responseWithFullError {
		server.errorToString = func(err error, _ string) string {
			return err.Error()
		}
	}

	return server
}

func (s *Server) Listen(addr string) error {
	var err error
	httpServer := &http.Server{Addr: addr, Handler: s}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logger.Info().Str("address", listener.Addr().String()).Msg("start listening")

	if s.options.tls {
		err = httpServer.ServeTLS(listener, s.options.tlsCertFile, s.options.tlsKeyFile)
	} else {
		err = httpServer.Serve(listener)
	}
	return err

}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	logger.Trace().Str("path", r.URL.Path).Str("method", r.Method).Msg("Got request")

	switch r.URL.Path {
	case "/sign":
		s.handleSigningRequest(w, r)
	default:
		http.NotFound(w, r)
	}

}

func (s *Server) acquireJob() bool {

	if s.options.maxConcurrentJobs <= 0 {
		return true
	}

	s.jobCounterMx.Lock()
	defer s.jobCounterMx.Unlock()

	if s.options.maxConcurrentJobs > 0 && s.jobCounter >= s.options.maxConcurrentJobs {
		return false
	}

	s.jobCounter++

	return true
}

func (s *Server) releaseJob() {
	if s.options.maxConcurrentJobs > 0 {
		s.jobCounterMx.Lock()
		s.jobCounter--
		s.jobCounterMx.Unlock()
	}
}

func (s *Server) parsePayload(w http.ResponseWriter, r *http.Request) (*signingPayload, bool) {

	var payload = &signingPayload{}
	var err error
	contentType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	switch contentType {
	case "application/json":
		err = json.NewDecoder(r.Body).Decode(payload)

	case "application/x-www-form-urlencoded":
		err = r.ParseForm()
		if err != nil {
			break
		}
		payload.ShortcutName = r.Form.Get("shortcutName")
		payload.Shortcut = r.Form.Get("shortcut")
	case "multipart/form-data":
		err = r.ParseMultipartForm(1 * MB)
		if err != nil {
			break
		}
		payload.ShortcutName = r.Form.Get("shortcutName")

		mFile, _, err := r.FormFile("shortcut")
		if err != nil {
			break
		}
		defer mFile.Close()
		content, err := io.ReadAll(mFile)
		if err != nil {
			break
		}

		payload.Shortcut = string(content)

	case "application/yaml":
		err = yaml.NewDecoder(r.Body).Decode(payload)
	case "application/x-plist":
		var data []byte
		data, err = io.ReadAll(r.Body)
		if err != nil {
			break
		}
		payload.Shortcut = string(data)

	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return nil, false
	}

	if errors.Is(err, &http.MaxBytesError{}) {
		http.Error(w, "Content too large", http.StatusRequestEntityTooLarge)
		return nil, false
	}

	if err != nil {
		logger.Error().Err(err).Msg("decoding payload")
		http.Error(w, s.errorToString(err, "Error parsing content"), http.StatusBadRequest)
		return nil, false
	}

	return payload, true

}

func (s *Server) prepareFilepaths(w http.ResponseWriter, r *http.Request) (string, string, func(), bool) {
	newUlid, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		logger.Error().Err(err).Msg("generating ulid")
		http.Error(w, s.errorToString(err, "Internal server error"), http.StatusInternalServerError)
		return "", "", nil, false
	}
	fileName := newUlid.String()

	if fileName == "" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return "", "", nil, false
	}

	dir, err := createTempDir(s.options.tempDir, fileName)
	if err != nil {
		logger.Error().Err(err).Msg("creating temp dir")
		http.Error(w, s.errorToString(err, "Internal server error"), http.StatusInternalServerError)
		return "", "", nil, false
	}

	unsignedShortcut := filepath.Join(dir, fileName+"_unsigned.shortcut")
	signedShortcut := filepath.Join(dir, fileName+".shortcut")
	cleanUpFunc := func() {
		err := os.RemoveAll(dir)
		if err != nil {
			logger.Error().Err(err).Str("dir", dir).Msg("removing temp dir")
		}
	}
	return unsignedShortcut, signedShortcut, cleanUpFunc, true
}

func (s *Server) verifyPayload(w http.ResponseWriter, r *http.Request, payload *signingPayload) bool {
	if payload.Shortcut == "" {
		http.Error(w, "Missing shortcut", http.StatusBadRequest)
		return false
	}

	return true
}

func (s *Server) handlePostSigningRequest(w http.ResponseWriter, r *http.Request) {

	if !s.acquireJob() {
		http.Error(w, "Too many concurrent jobs", http.StatusServiceUnavailable)
		return
	}

	defer s.releaseJob()

	r.Body = http.MaxBytesReader(w, r.Body, int64(s.options.maxContentSize))

	payload, ok := s.parsePayload(w, r)
	if !ok {
		return
	}

	ok = s.verifyPayload(w, r, payload)
	if !ok {
		return
	}

	unsignedShortcut, signedShortcut, cleanup, ok := s.prepareFilepaths(w, r)
	if !ok {
		return
	}

	defer cleanup()

	err := saveShortcut(unsignedShortcut, payload.Shortcut)
	if err != nil {
		logger.Error().Err(err).Msg("saving shortcut")
		http.Error(w, s.errorToString(err, "Internal server error"), http.StatusInternalServerError)
		return
	}

	signedShortcutContent, err := signShortcut(unsignedShortcut, signedShortcut)

	if err != nil {
		logger.Error().Err(err).Msg("signing shortcut")
		http.Error(w, s.errorToString(err, "Internal server error"), http.StatusInternalServerError)
		return
	}

	shortcutName := payload.ShortcutName
	if strings.TrimSpace(shortcutName) == "" {
		fileName, _, _ := strings.Cut(filepath.Base(unsignedShortcut), "_")
		shortcutName = fileName
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+shortcutName+".shortcut")
	w.WriteHeader(http.StatusOK)
	w.Write(signedShortcutContent)

}

func (s *Server) handleGetSigningRequest(w http.ResponseWriter, r *http.Request) {
	var templatesFs fs.FS
	if s.options.templateDir != "" {
		templatesFs = os.DirFS(s.options.templateDir)
	} else {
		templatesFs = templates
	}

	parsed, err := template.ParseFS(templatesFs, "**/*.html")
	if err != nil {
		http.Error(w, s.errorToString(err, "Internal server error"), http.StatusInternalServerError)
		return
	}
	err = parsed.ExecuteTemplate(w, "form.html", nil)
	if err != nil {
		http.Error(w, s.errorToString(err, "Internal server error"), http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "text/html")

}

func (s *Server) handleSigningRequest(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		s.handleGetSigningRequest(w, r)
	case "POST":
		s.handlePostSigningRequest(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}
