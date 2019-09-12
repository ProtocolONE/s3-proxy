package s3

import (
	"context"
	"crypto/tls"
	"github.com/Nerufa/go-shared/invoker"
	"github.com/Nerufa/go-shared/logger"
	"github.com/Nerufa/go-shared/provider"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

type S3 struct {
	ctx context.Context
	cfg Config
	*provider.AwareSet
}

// Delete
func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client().DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	return err
}

// Download
func (s *S3) Download(ctx context.Context, key string, w io.Writer) (n int64, err error) {
	// Get object from S3 storage
	resp, err := s.client().GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, err
	}
	n, err = io.Copy(w, resp.Body)
	if err != nil {
		return 0, err
	}
	return
}

// Upload
func (s *S3) Upload(ctx context.Context, key string, r io.Reader) (n int64, err error) {
	var (
		totalBytes int64
		refError   string
	)
	rs, err := s.syncPipe(r, &totalBytes, &refError)
	if err != nil {
		return 0, err
	}
	// Put object to S3 storage
	_, err = s.client().PutObjectWithContext(ctx, &s3.PutObjectInput{
		ACL:    &s.cfg.ACL,
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
		Body:   rs,
	})
	if refError != "" {
		return 0, errors.New(refError)
	}
	// S3 error handling
	if err != nil {
		return 0, err
	}
	return totalBytes, err
}

// syncPipeSeeker
type syncPipeSeeker struct {
	file        *os.File
	read, total int64
	err         error
	mu          sync.Mutex
}

// Read
func (s *syncPipeSeeker) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, err := s.file.Read(p)
	s.read += int64(l)
	if s.err != nil {
		return 0, err
	}
	if s.total != 0 && err == io.EOF && s.read == s.total {
		return l, err
	}
	return l, nil
}

// Seek
func (s *syncPipeSeeker) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, e := s.file.Seek(offset, whence)
	s.read = 0
	return i, e
}

// syncPipe
func (s *S3) syncPipe(r io.Reader, n *int64, refError *string) (io.ReadSeeker, error) {
	// Save to tmp file
	tmpFile, err := ioutil.TempFile(os.TempDir(), Prefix+"-")
	if err != nil {
		return nil, err
	}
	s.L().Debug("file created: %v", logger.Args(tmpFile.Name()))
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()
	out := &syncPipeSeeker{}
	// Copy bytes to tmp file to avoid memory allocation
	go func() {
		var err error
		*n, err = io.Copy(tmpFile, r)
		if err != nil {
			*refError = err.Error()
			out.err = err
			s.L().Debug("error occurred during copy to %v, err: %v", logger.Args(tmpFile.Name(), err.Error()))
		}
		out.total = *n
		// Try to close file
		_ = tmpFile.Close()
	}()
	// Read in parallel
	f, e := os.Open(tmpFile.Name())
	if e != nil {
		return nil, e
	}
	out.file = f
	return out, nil
}

// client
func (s *S3) client() *s3.S3 {
	sess := session.Must(session.NewSession(&aws.Config{
		HTTPClient:  s.httpClient(),
		Region:      aws.String(s.cfg.Region),
		Credentials: credentials.NewStaticCredentials(s.cfg.AccessKeyId, s.cfg.SecretAccessKey, s.cfg.Token),
		Endpoint:    aws.String(s.cfg.Endpoint),
	}))
	return s3.New(sess)
}

// httpClient
func (s *S3) httpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.cfg.InsecureSkipVerify},
	}
	return &http.Client{Transport: tr}
}

// Endpoint
func (s *S3) Endpoint() string {
	return s.cfg.Endpoint
}

// DownloadURL
func (s *S3) DownloadURL() string {
	return s.cfg.Endpoint + "/" + s.cfg.Bucket
}

// Config
type Config struct {
	Debug              bool `fallback:"shared.debug"`
	AccessKeyId        string
	SecretAccessKey    string
	Token              string
	Region             string
	Bucket             string
	Endpoint           string
	InsecureSkipVerify bool
	ACL                string
	invoker            *invoker.Invoker
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
func New(ctx context.Context, set provider.AwareSet, cfg *Config) *S3 {
	as := &set
	as.Logger = as.Logger.WithFields(logger.Fields{"service": Prefix})
	return &S3{
		ctx:      ctx,
		cfg:      *cfg,
		AwareSet: &set,
	}
}
