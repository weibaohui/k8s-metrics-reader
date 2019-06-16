package main

import (
	"fmt"
	"github.com/kubernetes/client-go/restmapper"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"net"
	"strings"
)

func config() (*rest.Config, error) {
	const (
		tokenFile  = "/Users/baohui/.kube/test-token"
		rootCAFile = "/Users/baohui/.kube/test-ca.crt"
	)
	host, port := "127.0.0.1", "6443"

	token, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	tlsClientConfig := rest.TLSClientConfig{}

	if _, err := certutil.NewPool(rootCAFile); err != nil {
		klog.Errorf("Expected to load root CA config from %s, but got err: %v", rootCAFile, err)
	} else {
		tlsClientConfig.CAFile = rootCAFile
	}

	return &rest.Config{
		// TODO: switch to using cluster DNS.
		Host:            "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: tlsClientConfig,
		BearerToken:     strings.TrimSpace(string(token)),
		BearerTokenFile: tokenFile,
	}, nil
}

func main() {
	// // set up the client config
	var clientConfig *rest.Config
	clientConfig, err := clientcmd.BuildConfigFromFlags("", "")
	clientConfig, err = config()
	clientset, err := kubernetes.NewForConfig(clientConfig)
	informers.NewSharedInformerFactory(clientset, 0)

	if err != nil {
		fmt.Printf("unable to construct lister client: %v", err)
	}
	m := "/apis/metrics.k8s.io/v1beta1/pods"
	raw, err := clientset.CoreV1().RESTClient().Get().RequestURI(m).DoRaw()
	if err != nil {
		fmt.Println("core", err.Error())
	}
	fmt.Println("core", len(string(raw)))

	bytes, err := clientset.RESTClient().Get().RequestURI(m).DoRaw()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("REST", len(string(bytes)))

	metricsClient := resourceclient.NewForConfigOrDie(clientConfig)
	metrics, err := metricsClient.PodMetricses("").List(metav1.ListOptions{})

	for _, m := range metrics.Items {
		podSum := int64(0)
		for _, c := range m.Containers {
			resValue, found := c.Usage[v1.ResourceName("memory")]
			if !found {
				break // containers loop
			}
			podSum += resValue.MilliValue()
		}

		fmt.Println(m.Name, podSum)

	}
	// Use a discovery client capable of being refreshed.
	cachedClient := cacheddiscovery.NewMemCacheClient(clientset.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)
	apiVersionsGetter := custom_metrics.NewAvailableAPIsGetter(clientset.Discovery())
	customMetricsClient := custom_metrics.NewForConfig(clientConfig, restMapper, apiVersionsGetter)

	custom_metrics_list, err := customMetricsClient.NamespacedMetrics("istio-system").
		GetForObjects(schema.GroupKind{Kind: "Pod"}, labels.Everything(), "spec_cpu_quota", labels.Everything())
	if err != nil {
		fmt.Println(err.Error())
	}
	if custom_metrics_list != nil && len(custom_metrics_list.Items) > 0 {

		for _, m := range custom_metrics_list.Items {
			fmt.Println(m.DescribedObject.Name,m.DescribedObject.Namespace,m.DescribedObject.Kind)
			fmt.Println(m.Metric.Name,m.Value,m.Value.MilliValue())
		}
	}

	select {}
}
