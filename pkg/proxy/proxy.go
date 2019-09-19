package proxy

import (
	"context"
	"github.com/Nerufa/go-shared/invoker"
	"github.com/Nerufa/go-shared/logger"
	"github.com/Nerufa/go-shared/provider"
	"github.com/ProtocolONE/s3-proxy/pkg/s3"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

// S3Proxy
type S3Proxy struct {
	storage     s3.Storage
	maxBodySize int64
	ctx         context.Context
	cfg         Config
	*provider.AwareSet
}

// Use
func (s *S3Proxy) Use(router *chi.Mux) {
	router.Use(s.cfg.Middleware...)
}

// Routers
func (s *S3Proxy) Routers(router *chi.Mux) {
	router.Post("/upload", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		id, err := uuid.NewRandom()
		if err != nil {
			s.L().Error("upload failed on generate UUID, error: %v", logger.Args(err))
			http.Error(rw, "500 Server Internal Error", http.StatusInternalServerError)
			return
		}
		log := s.L().WithFields(logger.Fields{"uuid": id})
		// Check on max length
		r.Body = http.MaxBytesReader(rw, r.Body, s.maxBodySize)
		if err := r.ParseMultipartForm(int64(s.cfg.MaxMemorySize)); err != nil {
			http.Error(rw, "413 Payload Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		// Get data from file multipart field
		file, fileHeader, err := r.FormFile(FormFile)
		if err != nil {
			http.Error(rw, "400 Bad Request", http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()
		// Detect mime type
		header := make([]byte, 512)
		_, err = file.ReadAt(header, 0)
		if err != nil && err != io.EOF {
			log.Error("read first 512 bytes to detect mime type failed, error: %v", logger.Args(err))
			http.Error(rw, "500 Server Internal Error", http.StatusInternalServerError)
			return
		}
		ft := http.DetectContentType(header)
		group := s.ResolveMimeTypeGroup(ft)
		if group == nil {
			// Try resolve mime type by fileHeader.Filename
			fallbackFileExt := filepath.Ext(fileHeader.Filename)
			ft =  mime.TypeByExtension(fallbackFileExt)
			group = s.ResolveMimeTypeGroup(ft)
			if group == nil {
				http.Error(rw, "415 Unsupported Media Type", http.StatusUnsupportedMediaType)
				return
			}
		}
		//
		var extension string
		exts, err := mime.ExtensionsByType(ft)
		if err != nil {
			log.Error("upload failed, error: %v", logger.Args(err))
			http.Error(rw, "500 Server Internal Error", http.StatusInternalServerError)
			return
		}
		if len(exts) == 0 {
			// Try get extension from fileHeader.Filename
			fallbackFileExt := filepath.Ext(fileHeader.Filename)
			if fallbackFileExt == "" {
				log.Error("not found extension for %v mime/type", logger.Args(ft))
				http.Error(rw, "422 Unprocessable Entity", http.StatusUnprocessableEntity)
				return
			}
			extension = fallbackFileExt
		} else {
			extension = exts[0]
		}
		limFile := http.MaxBytesReader(rw, file, int64(group.MaxUploadSize))
		fileName := id.String() + extension
		keyS3Path := group.Folder + "/" + fileName
		// Start uploading
		if _, err := s.storage.Upload(s.ctx, keyS3Path, limFile); err != nil {
			if err.Error() == "http: request body too large" {
				http.Error(rw, "413 Payload Too Large", http.StatusRequestEntityTooLarge)
				return
			}
			log.Error("upload failed, error: %v", logger.Args(err))
			http.Error(rw, "500 Server Internal Error", http.StatusInternalServerError)
			return
		}
		// OK Response
		render.JSON(rw, r, map[string]string{
			"file":          fileName,
			"relative_path": keyS3Path,
			"base_url":      s.cfg.BaseUrl,
		})
	}))
}

// ResolveMimeTypeGroup
func (s *S3Proxy) ResolveMimeTypeGroup(mimeType string) *Group {
	for _, group := range s.cfg.Groups {
		if funk.ContainsString(group.MimeTypes, mimeType) {
			return group
		}
	}
	if s.cfg.Defaults.AllowAnyMimeType {
		return &Group{
			MimeTypes:     []string{mimeType},
			MaxUploadSize: s.cfg.Defaults.MaxUploadSize,
			Folder:        s.cfg.Defaults.Folder,
		}
	}
	return nil
}

func (s *S3Proxy) init() {
	for _, group := range s.cfg.Groups {
		gSize := int64(group.MaxUploadSize)
		if gSize > s.maxBodySize {
			s.maxBodySize = gSize
		}
	}
	dSize := int64(s.cfg.Defaults.MaxUploadSize)
	if dSize > s.maxBodySize {
		s.maxBodySize = dSize
	}
}

// Group
type Group struct {
	MimeTypes     []string
	MaxUploadSize Size `fallback:"proxy.maxUploadSize"`
	Folder        string
}

// Defaults
type Defaults struct {
	AllowAnyMimeType bool
	MaxUploadSize    Size
	Folder           string
}

// Config
type Config struct {
	Debug         bool `fallback:"shared.debug"`
	BaseUrl       string
	Middleware    []func(http.Handler) http.Handler
	Groups        map[string]*Group
	Defaults      Defaults
	MaxMemorySize Size
	invoker       *invoker.Invoker
}

// OnReload
func (c *Config) OnReload(callback func(ctx context.Context)) {
	c.invoker.OnReload(callback)
}

// Reload
func (c *Config) Reload(ctx context.Context) {
	c.invoker.Reload(ctx)
}

// New
func New(ctx context.Context, set provider.AwareSet, cfg *Config, storage s3.Storage) *S3Proxy {
	as := &set
	as.Logger = as.Logger.WithFields(logger.Fields{"service": Prefix})
	s3p := &S3Proxy{
		storage:  storage,
		ctx:      ctx,
		cfg:      *cfg,
		AwareSet: &set,
	}
	s3p.init()
	return s3p
}
