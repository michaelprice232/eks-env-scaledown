package config

import (
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ScaleAction string

const (
	ScaleUp   ScaleAction = "ScaleUp"
	ScaleDown ScaleAction = "ScaleDown"
)

type Config struct {
	K8sClient kubernetes.Interface
	Action    ScaleAction
}

func (c Config) validateAction() error {
	switch c.Action {
	case ScaleUp, ScaleDown:
		return nil
	default:
		return fmt.Errorf("invalid Action: must be 'ScaleUp' or 'ScaleDown'. Ensure SCALE_ACTION envar is set correctly")
	}
}

func NewConfig() (Config, error) {
	var conf Config

	conf.Action = ScaleAction(os.Getenv("SCALE_ACTION"))
	err := conf.validateAction()
	if err != nil {
		return conf, fmt.Errorf("valiadting ScaleAction: %w", err)
	}

	kc, err := newK8sClient()
	if err != nil {
		return conf, fmt.Errorf("creating k8s client: %w", err)
	}
	conf.K8sClient = kc

	return conf, nil
}

func SetupLogging() {
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	var level log.Level

	switch logLevelStr {
	case "debug":
		level = log.LevelDebug
	case "info":
		level = log.LevelInfo
	case "warn", "warning":
		level = log.LevelWarn
	case "error":
		level = log.LevelError
	default:
		level = log.LevelInfo
	}

	handler := log.NewJSONHandler(os.Stdout, &log.HandlerOptions{Level: level})

	log.SetDefault(log.New(handler))
}

func newK8sClient() (*kubernetes.Clientset, error) {
	var client *kubernetes.Clientset
	var config *rest.Config
	var err error

	runningLocally := os.Getenv("RUNNING_LOCALLY")
	if runningLocally == "true" {
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
		if err != nil {
			return nil, fmt.Errorf("building K8s client config from the local host: %w", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("building K8s client config from the cluster: %w", err)
		}
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating K8s client: %w", err)
	}

	return client, nil
}
