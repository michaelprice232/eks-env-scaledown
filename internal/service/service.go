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

type K8sResource struct {
	Name           string
	Type           string
	Namespace      string
	ReplicaCount   int32
	Selector       string
	podsTerminated bool
}

type StartUpOrder map[int][]K8sResource
type Service struct {
	Conf         config.Config
	StartUpOrder StartUpOrder
}

func NewService(c config.Config) (*Service, error) {
	return &Service{Conf: c}, nil
}

func (s *Service) Run() error {
	switch s.Conf.Action {
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
	// todo: implement. Add a wait between each group to allow time for the MongoDB cluster to fully initialise
	log.Info("Scaling environment up")

	if err := s.BuildStartUpOrder(); err != nil {
		return fmt.Errorf("building startup order: %w", err)
	}

	scaleDownOrder := make([]int, 0, len(s.StartUpOrder))
	for order := range s.StartUpOrder {
		scaleDownOrder = append(scaleDownOrder, order)
	}

	sort.Sort(sort.IntSlice(scaleDownOrder))
	log.Debug("Scale up order", "order", scaleDownOrder)

	for _, order := range scaleDownOrder {
		log.Debug("Scaling up group", "group", order)
		if err := s.ScaleUpGroup(order); err != nil {
			return fmt.Errorf("scaling up group %d: %w", order, err)
		}
	}

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
			return fmt.Errorf("scaling down group %d: %w", order, err)
		}
	}

	return nil
}
