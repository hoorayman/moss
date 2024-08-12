package moss

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	moss_k8s "github.com/hoorayman/moss/pkg/k8s"
	"github.com/hoorayman/moss/pkg/metrics"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Moss struct {
	stopSignal    chan bool
	metricsServer *http.Server
	config        mossConfig
}

func NewMoss() *Moss {
	return &Moss{
		stopSignal: make(chan bool, 1),
		config:     readConfig(),
	}
}

func (moss *Moss) Start() {
	moss.metricsServer = metrics.StartMetricsServer(moss.config.prometheusEndpoint, moss.config.prometheusPort)
	clientset, err := moss.getClientSet()
	if err != nil {
		log.Fatalf("Error getting kubernetes clientset: %v", err)
	}
	resolver, err := moss_k8s.NewK8sIPResolver(clientset, moss.config.shouldResolveDns)
	if err != nil {
		log.Fatalf("Error creating resolver: %v", err)
	}
	err = resolver.StartWatching()
	if err != nil {
		log.Fatalf("Error watching cluster's state: %v", err)
	}

	// wait for resolver to populate
	time.Sleep(10 * time.Second)

	go func() {
		for {
			select {
			case <-moss.stopSignal:
				return
			}
		}
	}()
}

func (moss *Moss) Stop() {
	log.Print("Stopping Moss...")
	moss.stopSignal <- true
	err := moss.metricsServer.Shutdown(context.Background())
	if err != nil {
		log.Printf("Error shutting Prometheus server down: %v", err)
	}
}

func (moss *Moss) getClientSet() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	// Check if running inside the cluster
	if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); exists {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// Use external cluster config (e.g., from kubeconfig file)
		kubeconfig := filepath.Join(
			os.Getenv("HOME"), ".kube", "config",
		)
		if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
			kubeconfig = kubeconfigEnv
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
