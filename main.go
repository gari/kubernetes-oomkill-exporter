package main

import (
	"flag"
	"net/http"
	"os"
	"regexp"

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
	kubernetesCounterVec      *prometheus.CounterVec
	metricsAddr               string
	kubeAPI                   bool
	nodeName                  string
	prometheusContainerLabels = map[string]string{
		"io.kubernetes.container.name": "container_name",
		"io.kubernetes.pod.namespace":  "namespace",
		"io.kubernetes.pod.uid":        "pod_uid",
		"io.kubernetes.pod.name":       "pod_name",
	}
	// dockerClient *docker_client.Client
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

	labels := []string{"pod_name", "node_name", "namespace", "unit", "container_name"}
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
	// fmt.Println(logCh)

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
					containerLabels := pod.Labels
					containerLabels["namespace"] = pod.Namespace
					containerLabels["pod_name"] = pod.Name
					containerLabels["node_name"] = nodeName
					for _, c := range pod.Status.ContainerStatuses {
						if string(c.ContainerID) == foundContainerID {
							containerLabels["container_name"] = c.Name
						}

					}
					prometheusCount(containerLabels)
				}
			}
			if err != nil {
				panic(err.Error())
			}
		}
		// } else {
		// if containerID != "" {
		// 	container, err := getContainer(containerID, dockerClient)
		// 	if err != nil {
		// 		glog.Warningf("Could not get container %s for pod %s: %v", containerID, podUID, err)
		// 	} else {
		// 		prometheusCount(container.Config.Labels)
		// 	}
		// }
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
