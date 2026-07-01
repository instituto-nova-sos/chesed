package service

import (
	"context"
	"fmt"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// ServiceTypeRepository defines the interface for service type persistence.
//
//nolint:revive // domain-meaningful name used across handler/cmd/repository; rename deferred
type ServiceTypeRepository interface {
	ListActive(ctx context.Context) ([]domain.ServiceType, error)
}

// ServiceTypeService handles service type operations.
//
//nolint:revive // domain-meaningful name used across handler/cmd; rename deferred
type ServiceTypeService struct {
	repo ServiceTypeRepository
}

// NewServiceTypeService creates a new ServiceTypeService.
func NewServiceTypeService(repo ServiceTypeRepository) *ServiceTypeService {
	return &ServiceTypeService{repo: repo}
}

// ListActive returns all active service types.
func (s *ServiceTypeService) ListActive(ctx context.Context) ([]domain.ServiceType, error) {
	types, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("serviceTypeService.ListActive: %w", err)
	}
	return types, nil
}
