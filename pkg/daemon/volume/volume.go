package volume

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rancher/k3c/pkg/client"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Volume string

const (
	HostPathVolumeType = Volume("hostPath")
	NamedVolumeType    = Volume("named")
	Ephemeral          = Volume("ephemeral")
)

var (
	DefaultVolumeDir = "/var/lib/rancher/k3c/volumes"
	re               = regexp.MustCompile("^[-a-zA-Z0-9]+$")
	hexRe            = regexp.MustCompile("^[a-f0-9]{32}$")
)

type Manager struct {
	baseDir string
}

func New(baseDir string) (*Manager, error) {
	return &Manager{
		baseDir: baseDir,
	}, os.MkdirAll(baseDir, 0700)
}

func PathToType(name string) (string, Volume) {
	if strings.HasPrefix(name, DefaultVolumeDir) {
		name = filepath.Base(name)
		if hexRe.MatchString(name) {
			return name, Ephemeral
		}
		return name, NamedVolumeType
	}
	d := md5.Sum([]byte(name))
	return hex.EncodeToString(d[:]), HostPathVolumeType
}

func (m *Manager) Setup(ctx context.Context, mount *v1alpha2.Mount) (*v1alpha2.Mount, error) {
	var err error

	mnt := *mount
	if mount.HostPath == "" {
		v, err := m.CreateVolume(ctx, "")
		if err != nil {
			return nil, err
		}
		mnt.HostPath, err = m.Resolve(v.ID)
		return &mnt, err
	} else if !strings.HasPrefix(mnt.HostPath, "/") {
		mnt.HostPath, err = m.Resolve(mnt.HostPath)
		return &mnt, err
	}

	return mount, nil
}

func (m *Manager) Resolve(id string) (string, error) {
	if !re.MatchString(id) {
		return "", fmt.Errorf("invalid volume name")
	}
	return filepath.Join(m.baseDir, id), nil
}

func (m *Manager) GetVolume(name string) (*client.Volume, error) {
	if !re.MatchString(name) {
		return nil, fmt.Errorf("invalid volume name")
	}

	path := filepath.Join(m.baseDir, name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return &client.Volume{
		ID: name,
	}, nil
}

func (m *Manager) CreateVolume(ctx context.Context, name string) (*client.Volume, error) {
	if name == "" {
		id := uuid.New()
		name = hex.EncodeToString(id[:])
	}

	if !re.MatchString(name) {
		return nil, fmt.Errorf("invalid volume name")
	}

	path := filepath.Join(m.baseDir, name)
	if err := os.Mkdir(path, 0755); err != nil {
		return nil, err
	}

	return &client.Volume{
		ID: name,
	}, nil
}

func (m *Manager) RemoveVolume(ctx context.Context, name string, force bool) error {
	_, err := m.GetVolume(name)
	if os.IsNotExist(err) {
		return nil
	}
	p, err := m.Resolve(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(p)
}

func (m *Manager) ListVolumes(ctx context.Context) (result []client.Volume, err error) {
	files, err := ioutil.ReadDir(m.baseDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		result = append(result, client.Volume{
			ID: file.Name(),
		})
	}

	return
}
