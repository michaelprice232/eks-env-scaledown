// Package config loads runtime configuration from environment variables and
// builds the Kubernetes clients used by the rest of the application.
package config

import (
	"fmt"
	log "log/slog"
	"os"
	"strconv"
	"strings"

	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// ScaleAction defines whether the environment should be scaled up or down.
type ScaleAction string

const (
	// ScaleUp restores workloads to their previous replica counts.
	ScaleUp ScaleAction = "ScaleUp"
	// ScaleDown scales workloads to zero replicas.
	ScaleDown ScaleAction = "ScaleDown"
)

// Config holds the runtime configuration and Kubernetes clients for the application.
type Config struct {
	K8sClient        kubernetes.Interface
	K8sDynamicClient dynamic.Interface
	Action           ScaleAction
	SuspendCronJob   bool
	SuspendKeda      bool
}

func (c Config) validateAction() error {
	switch c.Action {
	case ScaleUp, ScaleDown:
		return nil
	default:
		return fmt.Errorf("invalid Action: must be 'ScaleUp' or 'ScaleDown'. Ensure SCALE_ACTION envar is set correctly")
	}
}

// parseBoolEnv reads a boolean environment variable, returning def when the variable
// is unset or cannot be parsed as a boolean.
func parseBoolEnv(key string, def bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return def
	}

	parsed, err := strconv.ParseBool(val)
	if err != nil {
		log.Warn("Problem parsing boolean env var. Using default", "key", key, "value", val, "default", def)
		return def
	}

	return parsed
}

// NewConfig builds a Config from environment variables and initialises the Kubernetes clients.
func NewConfig() (Config, error) {
	var conf Config

	conf.Action = ScaleAction(os.Getenv("SCALE_ACTION"))
	err := conf.validateAction()
	if err != nil {
		return conf, fmt.Errorf("validating ScaleAction: %w", err)
	}

	// Whether to disable CronJobs (except the one managing this app) during the scaledown. Default to enable
	conf.SuspendCronJob = parseBoolEnv("SUSPEND_CRONJOB", true)

	// Whether to disable Keda ScaledObjects during the scaledown. Default to disabled
	conf.SuspendKeda = parseBoolEnv("SUSPEND_KEDA_SCALED_OBJECTS", false)

	kc, dc, err := newK8sClients()
	if err != nil {
		return conf, fmt.Errorf("creating k8s clients: %w", err)
	}
	conf.K8sClient = kc
	conf.K8sDynamicClient = dc

	return conf, nil
}

// SetupLogging configures the default structured logger using the LOG_LEVEL environment variable.
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

func newK8sClients() (*kubernetes.Clientset, *dynamic.DynamicClient, error) {
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
			return nil, nil, fmt.Errorf("building K8s client config from the local host: %w", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("building K8s client config from the cluster: %w", err)
		}
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("creating K8s client: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("creating K8s dynamic client: %w", err)
	}

	return client, dynClient, nil
}
