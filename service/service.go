package service

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/runityru/cephctl/ceph"
	"github.com/runityru/cephctl/differ"
	"github.com/runityru/cephctl/models"
	clusterHealth "github.com/runityru/cephctl/service/cluster_health"
)

type Service interface {
	ApplyCephConfig(ctx context.Context, cfg models.CephConfig) error
	DiffCephConfig(ctx context.Context, cfg models.CephConfig) ([]models.CephConfigDifference, error)
	CheckClusterHealth(ctx context.Context, checks []clusterHealth.ClusterHealthCheck) ([]models.ClusterHealthIndicator, error)
	DumpConfig(ctx context.Context) (models.CephConfig, error)
}

type service struct {
	c ceph.Ceph
	d differ.Differ
}

func New(c ceph.Ceph, d differ.Differ) Service {
	return &service{
		c: c,
		d: d,
	}
}

func (s *service) ApplyCephConfig(ctx context.Context, cfg models.CephConfig) error {
	changes, err := s.DiffCephConfig(ctx, cfg)
	if err != nil {
		return errors.Wrap(err, "error comparing current and desired configuration")
	}

	log.WithFields(log.Fields{
		"component": "service",
	}).Tracef("changelog: %#v", changes)

	for _, change := range changes {
		switch change.Kind {
		case models.CephConfigDifferenceKindRemove:
			if err := s.c.RemoveCephConfigOption(ctx, change.Section, change.Key); err != nil {
				return err
			}
		case models.CephConfigDifferenceKindAdd, models.CephConfigDifferenceKindChange:
			if err := s.c.ApplyCephConfigOption(ctx, change.Section, change.Key, *change.Value); err != nil {
				return err
			}
		default:
			log.Warnf("unexpected change kind: %s", change.Kind)
		}
	}
	return nil
}

func (s *service) CheckClusterHealth(ctx context.Context, checks []clusterHealth.ClusterHealthCheck) ([]models.ClusterHealthIndicator, error) {
	cr, err := s.c.ClusterReport(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving cluster status")
	}

	devices, err := s.c.ListDevices(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving device list")
	}

	cr.Devices = devices

	indicators := []models.ClusterHealthIndicator{}
	for _, checkFunc := range checks {
		indicator, err := checkFunc(ctx, cr)
		if err != nil {
			return nil, err
		}

		indicators = append(indicators, indicator)
	}

	return indicators, nil
}

func (s *service) DiffCephConfig(ctx context.Context, cfg models.CephConfig) ([]models.CephConfigDifference, error) {
	src, err := s.c.DumpConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving current configuration")
	}

	return s.d.DiffCephConfig(ctx, src, cfg)
}

func (s *service) DumpConfig(ctx context.Context) (models.CephConfig, error) {
	return s.c.DumpConfig(ctx)
}
