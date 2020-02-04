package mock

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/cybozu-go/sabakan/v2"
)

type imageData struct {
	kernel []byte
	initrd []byte
}

type imageDriver struct {
	mu     sync.Mutex
	index  sabakan.ImageIndex
	images map[string]imageData
}

func newImageDriver() *imageDriver {
	return &imageDriver{
		images: make(map[string]imageData),
	}
}

func (d *imageDriver) GetIndex(ctx context.Context, os string) (sabakan.ImageIndex, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	copied := make(sabakan.ImageIndex, len(d.index))
	copy(copied, d.index)
	for _, i := range copied {
		i.Exists = true
	}
	return copied, nil
}

func (d *imageDriver) GetInfoAll(ctx context.Context) ([]*sabakan.Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	images := make([]*sabakan.Image, len(d.index))
	for i := range d.index {
		images[i] = d.index[i]
	}

	return images, nil
}

func (d *imageDriver) Upload(ctx context.Context, os, id string, r io.Reader) error {
	d.mu.Lock()
	defer func() {
		d.mu.Unlock()
		io.Copy(ioutil.Discard, r)
	}()

	if os != "coreos" {
		return errors.New("mock driver supports only coreos")
	}

	img := d.index.Find(id)
	if img != nil {
		return sabakan.ErrConflicted
	}

	var kernel, initrd []byte

	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch hdr.Name {
		case sabakan.ImageKernelFilename:
			b := new(bytes.Buffer)
			_, err := io.Copy(b, tr)
			if err != nil {
				return err
			}
			kernel = b.Bytes()

		case sabakan.ImageInitrdFilename:
			b := new(bytes.Buffer)
			_, err := io.Copy(b, tr)
			if err != nil {
				return err
			}
			initrd = b.Bytes()

		default:
			return sabakan.ErrBadRequest
		}
	}

	if kernel == nil || initrd == nil {
		return sabakan.ErrBadRequest
	}

	d.images[id] = imageData{kernel, initrd}
	d.index, _ = d.index.Append(&sabakan.Image{
		ID:   id,
		Date: time.Now().UTC(),
		Size: int64(len(kernel) + len(initrd)),
	})

	return nil
}

func (d *imageDriver) Download(ctx context.Context, os, id string, out io.Writer) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if os != "coreos" {
		return errors.New("mock driver supports only coreos")
	}

	img := d.index.Find(id)
	if img == nil {
		return sabakan.ErrNotFound
	}
	data := d.images[id]

	tw := tar.NewWriter(out)

	// kernel
	hdr := &tar.Header{
		Name: sabakan.ImageKernelFilename,
		Mode: 0644,
		Size: int64(len(data.kernel)),
	}
	err := tw.WriteHeader(hdr)
	if err != nil {
		return err
	}
	_, err = tw.Write(data.kernel)
	if err != nil {
		return err
	}

	// initrd
	hdr = &tar.Header{
		Name: sabakan.ImageInitrdFilename,
		Mode: 0644,
		Size: int64(len(data.initrd)),
	}
	err = tw.WriteHeader(hdr)
	if err != nil {
		return err
	}
	_, err = tw.Write(data.initrd)
	if err != nil {
		return err
	}

	return tw.Close()
}

func (d *imageDriver) Delete(ctx context.Context, os, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if os != "coreos" {
		return errors.New("mock driver supports only coreos")
	}

	img := d.index.Find(id)
	if img == nil {
		return sabakan.ErrNotFound
	}

	d.index = d.index.Remove(id)
	delete(d.images, id)
	return nil
}

func (d *imageDriver) ServeFile(ctx context.Context, os, filename string,
	f func(modtime time.Time, content io.ReadSeeker)) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if os != "coreos" {
		return errors.New("mock driver supports only coreos")
	}

	if len(d.index) == 0 {
		return sabakan.ErrNotFound
	}

	// the newest image
	img := d.index[len(d.index)-1]
	data := d.images[img.ID]

	switch filename {
	case sabakan.ImageKernelFilename:
		f(img.Date, bytes.NewReader(data.kernel))
	case sabakan.ImageInitrdFilename:
		f(img.Date, bytes.NewReader(data.initrd))
	default:
		return sabakan.ErrNotFound
	}

	return nil
}
