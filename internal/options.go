package internal

import (
	"os"
	"path/filepath"
)

type serverOptions struct {
	tempDir               string
	tls                   bool
	tlsCertFile           string
	tlsKeyFile            string
	maxContentSize        int
	responseWithFullError bool
	maxConcurrentJobs     int
	maxFilenameLength     int
	createRandomFilenames bool
}

type ServerOption func(*serverOptions) error

func TempDir(dir string) ServerOption {
	return func(so *serverOptions) error {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		_, err = os.Stat(abs)

		if err != nil {
			return err
		}

		so.tempDir = abs

		return nil
	}

}

func CreateRandomFilenames(createRandomFilenames bool) ServerOption {
	return func(so *serverOptions) error {
		so.createRandomFilenames = createRandomFilenames
		return nil
	}
}

func MaxFilenameLength(maxFilenameLength int) ServerOption {
	return func(so *serverOptions) error {
		so.maxFilenameLength = maxFilenameLength
		return nil
	}
}

func EnableFullErrorsRespones() ServerOption {
	return func(so *serverOptions) error {
		so.responseWithFullError = true
		return nil
	}
}

func EnableTls(tlsCertPath, tlsKeyPath string) ServerOption {
	NotImplemented(&logger)

	return func(so *serverOptions) error {
		return nil
	}
}

func MaxContentSize(maxContentSize int) ServerOption {
	return func(so *serverOptions) error {
		so.maxContentSize = maxContentSize
		return nil
	}
}
