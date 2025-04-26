package service

import (
	"fmt"
	log "log/slog"
	"sort"

	"github.com/michaelprice232/eks-env-scaledown/config"
)

type K8sResource struct {
	Name         string
	Type         string
	Namespace    string
	ReplicaCount int32
}

type StartUpOrder map[int][]K8sResource
type Service struct {
	conf         config.Config
	StartUpOrder StartUpOrder
}

func NewService(c config.Config) (*Service, error) {

	return &Service{conf: c}, nil
}

func (s *Service) Run() error {
	switch s.conf.Action {
	case config.ScaleUp:
		if err := s.EnvScaleUp(); err != nil {
			return fmt.Errorf("scaling environment up: %w", err)
		}
	case config.ScaleDown:
		if err := s.EnvScaleDown(); err != nil {
			return fmt.Errorf("scaling environment down: %w", err)
		}
	default:
		return fmt.Errorf("invalid ScaleAction detected. Must be 'ScaleUp' or 'ScaleDown'")
	}

	return nil
}

func (s *Service) EnvScaleUp() error {
	log.Info("Scaling environment up")

	return nil
}

func (s *Service) EnvScaleDown() error {
	log.Info("Scaling environment down")

	if err := s.BuildStartUpOrder(); err != nil {
		return fmt.Errorf("building startup order: %w", err)
	}

	scaleDownOrder := make([]int, 0, len(s.StartUpOrder))
	for order := range s.StartUpOrder {
		scaleDownOrder = append(scaleDownOrder, order)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(scaleDownOrder)))
	log.Debug("Scale down order", "order", scaleDownOrder)

	for _, order := range scaleDownOrder {
		log.Debug("Scaling down group", "group", order)
		if err := s.ScaleDownGroup(order); err != nil {
			return fmt.Errorf("scaling down group %d: %w", 1, err)
		}
	}

	return nil
}
