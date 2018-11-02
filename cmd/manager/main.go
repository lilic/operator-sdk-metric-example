package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/example-inc/metric-operator/pkg/apis"
	"github.com/example-inc/metric-operator/pkg/controller"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	ksmetrics "k8s.io/kube-state-metrics/pkg/metrics"
)

func printVersion() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()
	flag.Parse()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("failed to get watch namespace: %v", err)
	}

	// TODO: Expose metrics port after SDK uses controller-runtime's dynamic client
	//sdk.ExposeMetricsPort()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	uc := metrics.NewForConfig(cfg)
	resource := "metric.example.com/v1alpha1"
	kind := "MetricService"
	c := metrics.NewCollectors(uc, []string{"default"}, resource, kind, GenerateStore)
	prometheus.MustRegister(c)

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
	}

	log.Print("Starting the Cmd.")

	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}

var (
	descMemStatusReplicas = ksmetrics.NewMetricFamilyDef(
		"metric_service_info",
		"The information of the operator instance.",
		[]string{"namespace", "metric-service"},
		nil,
	)
)

func GenerateStore(obj interface{}) []*ksmetrics.Metric {
	ms := []*ksmetrics.Metric{}
	crdp := obj.(*unstructured.Unstructured)

	crd := *crdp

	instances := float64(1)
	// get spec.replicas
	lv := []string{crd.GetNamespace(), crd.GetName()}
	m, err := ksmetrics.NewMetric(descMemStatusReplicas.Name, descMemStatusReplicas.LabelKeys, lv, instances)
	if err != nil {
		fmt.Println(err)
		return ms
	}
	ms = append(ms, m)
	fmt.Println("generate store")
	return ms
}
