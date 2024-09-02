package fs

import (
	io_fs "io/fs"
)

type apiDirEntry struct {
	info io_fs.FileInfo
}

func (d *apiDirEntry) Name() string {
	return d.info.Name()
}

func (d *apiDirEntry) IsDir() bool {
	return d.info.IsDir()
}

func (d *apiDirEntry) Type() io_fs.FileMode {
	return d.info.Mode()
}

func (d *apiDirEntry) Info() (io_fs.FileInfo, error) {
	return d.info, nil
}
