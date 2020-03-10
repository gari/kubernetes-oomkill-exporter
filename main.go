package main

import (
	"flag"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/kmsg"
	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
)

var (
	kmesgRE        = regexp.MustCompile(`/pod(\w+-\w+-\w+-\w+-\w+)/([a-f0-9]+) killed as a result of limit of /kubepods`)
	kmesgREkernel5 = regexp.MustCompile(`/pod(\w+-\w+-\w+-\w+-\w+)/([a-f0-9]+),task`)
)

var (
	kubernetesCounterVec *prometheus.CounterVec
	metricsAddr          string
	kubeAPI              bool
	nodeName             string
)

func init() {
	// var err error
	flag.StringVar(&metricsAddr, "listen-address", ":9102", "The address to listen on for HTTP requests.")
	flag.BoolVar(&kubeAPI, "kube-api", false, "Use of kube api instead of docker socket")
}

func main() {
	flag.Parse()

	nodeName = os.Getenv("NODE_NAME")
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	labels := []string{
		"pod_name",
		"node_name",
		"namespace",
		"container_name",
		"pod_uuid",
		"unit",
	}
	// labels =

	kubernetesCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "klog_pod_oomkill",
		Help: "Extract metrics for OOMKilled pods from kernel log",
	}, labels)

	prometheus.MustRegister(kubernetesCounterVec)

	go func() {
		glog.Info("Starting prometheus metrics")
		http.Handle("/metrics", promhttp.Handler())
		glog.Warning(http.ListenAndServe(metricsAddr, nil))
	}()

	kmsgWatcher := kmsg.NewKmsgWatcher(types.WatcherConfig{Plugin: "kmsg"})
	logCh, err := kmsgWatcher.Watch()

	if err != nil {
		glog.Fatal("Could not create log watcher")
	}

	for log := range logCh {
		foundPodUID, foundContainerID := getContainerIDFromLog(log.Message)
		if foundContainerID != "" && foundPodUID != "" {
			glog.Info("foundContainerID: " + foundContainerID)
			glog.Info("foundPodUID: " + foundPodUID)
			glog.Info("Try to use kube api to find pod")
			pods, _ := clientset.CoreV1().Pods("").List(metav1.ListOptions{
				FieldSelector: "spec.nodeName=" + nodeName,
			})

			for _, pod := range pods.Items {

				if string(pod.GetUID()) == foundPodUID {
					glog.Info("Success, found pod by uiid")
					glog.V(9).Infoln(pod)
					containerLabels := make(map[string]string)
					containerLabels["namespace"] = pod.Namespace
					containerLabels["pod_name"] = pod.Name
					containerLabels["pod_uuid"] = foundPodUID
					containerLabels["node_name"] = nodeName
					if _, ok := pod.Labels["unit"]; ok {
						containerLabels["unit"] = pod.Labels["unit"]
					} else {
						containerLabels["unit"] = ""
					}
					for _, c := range pod.Status.ContainerStatuses {
						glog.V(9).Infoln(c)
						if strings.Contains(string(c.ContainerID), foundContainerID) {
							containerLabels["container_name"] = c.Name
						}

					}
					if _, ok := containerLabels["container_name"]; !ok {
						containerLabels["container_name"] = ""
					}
					prometheusCount(containerLabels)
				}
			}
			if err != nil {
				panic(err.Error())
			}
		}
	}
}

func getContainerIDFromLog(log string) (string, string) {
	// fmt.Println("Log: " + log)
	if matches := kmesgRE.FindStringSubmatch(log); matches != nil {
		return matches[1], matches[2]
	}
	if matches := kmesgREkernel5.FindStringSubmatch(log); matches != nil {
		return matches[1], matches[2]
	}
	return "", ""
}

func prometheusCount(containerLabels map[string]string) {
	var counter prometheus.Counter
	var err error

	glog.V(5).Infof("Labels: %v\n", containerLabels)
	counter, err = kubernetesCounterVec.GetMetricWith(containerLabels)

	if err != nil {
		glog.Warning(err)
	} else {
		counter.Add(1)
	}
}
