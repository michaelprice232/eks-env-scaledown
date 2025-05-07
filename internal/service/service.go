package service

import (
	"fmt"
	log "log/slog"
	"sort"

	"github.com/michaelprice232/eks-env-scaledown/config"
)

const (
	StartupOrderAnnotationKey         = "eks-env-scaledown/startup-order"
	OriginalReplicasAnnotationKey     = "eks-env-scaledown/original-replicas"
	UpdatedAtAnnotationKey            = "eks-env-scaledown/updated-at"
	DefaultStartUpGroup           int = 100
)

type k8sResource struct {
	Name                string
	ResourceType        string
	Namespace           string
	ReplicaCount        int32
	Selector            string
	podsTerminated      bool
	podsUpdatedAndReady bool
}

type startUpOrder map[int][]k8sResource
type Service struct {
	conf         config.Config
	startUpOrder startUpOrder
}

func NewService(c config.Config) (*Service, error) {
	return &Service{conf: c}, nil
}

func (s *Service) Run() error {
	switch s.conf.Action {
	case config.ScaleUp:
		if err := s.envScaleUp(); err != nil {
			return fmt.Errorf("scaling environment up: %w", err)
		}
	case config.ScaleDown:
		if err := s.envScaleDown(); err != nil {
			return fmt.Errorf("scaling environment down: %w", err)
		}
	default:
		return fmt.Errorf("invalid ScaleAction detected. Must be 'ScaleUp' or 'ScaleDown'")
	}

	return nil
}

func (s *Service) envScaleUp() error {
	log.Info("Scaling environment up")

	if err := s.buildStartUpOrder(); err != nil {
		return fmt.Errorf("building startup order: %w", err)
	}

	scaleOrder := make([]int, 0, len(s.startUpOrder))
	for order := range s.startUpOrder {
		scaleOrder = append(scaleOrder, order)
	}

	sort.Sort(sort.IntSlice(scaleOrder))
	log.Debug("Scale up order", "order", scaleOrder)

	for _, order := range scaleOrder {
		log.Info("Scaling up group", "group", order)
		if err := s.scaleUpGroup(order); err != nil {
			return fmt.Errorf("scaling up group %d: %w", order, err)
		}
	}

	return nil
}

func (s *Service) envScaleDown() error {
	log.Info("Scaling environment down")

	if err := s.buildStartUpOrder(); err != nil {
		return fmt.Errorf("building startup order: %w", err)
	}

	scaleOrder := make([]int, 0, len(s.startUpOrder))
	for order := range s.startUpOrder {
		scaleOrder = append(scaleOrder, order)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(scaleOrder)))
	log.Debug("Scale down order", "order", scaleOrder)

	for _, order := range scaleOrder {
		log.Info("Scaling down group", "group", order)
		if err := s.scaleDownGroup(order); err != nil {
			return fmt.Errorf("scaling down group %d: %w", order, err)
		}
	}

	if err := s.terminateStandalonePods(); err != nil {
		log.Info("Terminating standalone pods")
		return fmt.Errorf("terminating standalone pods: %w", err)
	}

	return nil
}
