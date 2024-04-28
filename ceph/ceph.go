package ceph

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	cephModels "github.com/teran/cephctl/ceph/models"
	"github.com/teran/cephctl/models"
)

type Ceph interface {
	ApplyCephConfigOption(ctx context.Context, section, key, value string) error
	DumpConfig(ctx context.Context) (models.CephConfig, error)
	RemoveCephConfigOption(ctx context.Context, section, key string) error
}

type ceph struct {
	binaryPath string
}

func New(binaryPath string) Ceph {
	return &ceph{
		binaryPath: binaryPath,
	}
}

func (c *ceph) ApplyCephConfigOption(ctx context.Context, section, key, value string) error {
	args := []string{"config", "set", section, key, value}

	log.Tracef("preparing to run %s %s", c.binaryPath, strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error applying configuration")
	}
	return nil
}

func (c *ceph) DumpConfig(ctx context.Context) (models.CephConfig, error) {
	cfg := []cephModels.ConfigOption{}
	buf := &bytes.Buffer{}

	args := []string{"config", "dump", "--format=json"}

	log.Tracef("preparing to run %s %s", c.binaryPath, strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "error running command")
	}
	log.Tracef("command output: `%s`", buf.String())

	if err := json.Unmarshal(buf.Bytes(), &cfg); err != nil {
		return nil, errors.Wrap(err, "error decoding response")
	}

	out := make(models.CephConfig)
	for _, v := range cfg {
		if _, ok := out[v.Section]; !ok {
			out[v.Section] = make(map[string]string)
		}

		out[v.Section][v.Name] = v.Value
	}

	return out, nil
}

func (c *ceph) RemoveCephConfigOption(ctx context.Context, section, key string) error {
	args := []string{"config", "rm", section, key}

	log.Tracef("preparing to run %s %s", c.binaryPath, strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "error applying configuration")
	}
	return nil
}
