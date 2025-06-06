package config

import (
	"fmt"
	log "log/slog"
	"os"
	"strconv"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

type ScaleAction string

const (
	ScaleUp   ScaleAction = "ScaleUp"
	ScaleDown ScaleAction = "ScaleDown"
)

type Config struct {
	K8sClient      kubernetes.Interface
	Action         ScaleAction
	SuspendCronJob bool
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
		return conf, fmt.Errorf("validating ScaleAction: %w", err)
	}

	// Whether to disable CronJobs (except the one managing this app) during the scaledown. Default to enable
	suspendCronJobs := os.Getenv("SUSPEND_CRONJOB")
	if suspendCronJobs == "" {
		conf.SuspendCronJob = true
	} else {
		suspend, err := strconv.ParseBool(suspendCronJobs)
		if err != nil {
			log.Warn("Problem parsing SUSPEND_CRONJOB into a boolean. Defaulting to true")
			conf.SuspendCronJob = true
		} else {
			conf.SuspendCronJob = suspend
		}
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

	// Use a context from the local kubeconfig file if set (running locally), otherwise expect to be running in the cluster itself
	k8sContext := os.Getenv("KUBE_CONTEXT")
	if k8sContext != "" {
		log.Info("Using local kubeconfig context", "context", k8sContext)
		localContextFile := filepath.Join(homedir.HomeDir(), ".kube", "config")
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: localContextFile}
		overrides := &clientcmd.ConfigOverrides{CurrentContext: k8sContext}
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
		config, err = clientConfig.ClientConfig()
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
