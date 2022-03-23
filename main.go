package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kyaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/tools/clientcmd"
)

var filePath = flag.String("f", "", "The file path of pod yaml, default is empty")
var podNumber = flag.Int("n", 10, "Number of pods created, At least >= 1")
var goroutineNumber = flag.Int("g",10,"Number of concurrent goroutines, At least >1")

var podTmplate string = `apiVersion: v1
kind: Pod
metadata:
  name: calico-macvlan
  annotations:
    k8s.v1.cni.cncf.io/networks: kube-system/macvlan
spec:
  containers:
  - name: hello-v2
    image: cyclinder/hello:v2
    imagePullPolicy: IfNotPresent
    ports:
    - containerPort: 8080
    args:
    - podIp
    - hostName
`

func homeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return ""
}

// createPod
func createPod(number int, pod *v1.Pod, clientSet *kubernetes.Clientset) {
	if pod != nil {
		klog.Infof("the basic info of pod: %v", pod)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ch := make(chan int)
		wg := sync.WaitGroup{}
		wg.Add(*goroutineNumber)
		for i := 0; i < *goroutineNumber; i++ {
			go func(i int) {
				klog.V(5).Infof("the number %v goroutine starts working...", i)
				defer klog.V(5).Infof("the number %v goroutine stop working...", i)
				defer wg.Done()
				for idx := range ch {
				 	tmpP := *pod
				 	tmpP.Name = fmt.Sprintf("%s%v", tmpP.Name, idx)
					_, err := clientSet.CoreV1().Pods("default").Create(ctx, &tmpP, metav1.CreateOptions{})
					if err != nil {
						klog.Errorf("(%v goroutine create pod(%s) failed: %v",i, tmpP.Name, err)
					}
					 klog.Infof("(%v goroutine create pod(%s) success",i, tmpP.Name)
				}

			}(i)
		}

		for i := 0; i < number; i++ {
			ch <- i
		}
		close(ch)
		wg.Wait()
		return
	}
	klog.Infof("Pod is unexpected to be nil, No Pods were created")
}

func parseFileToPod(fp string, pod *v1.Pod) {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		klog.Errorf("read pod file path failed: %v", err)
		return
	}
	jsonData, err := yaml.YAMLToJSON(b)
	if err != nil {
		klog.Errorf("yaml tp json failed: %v", err)
		return
	}
	if err = json.Unmarshal(jsonData, pod); err != nil {
		klog.Errorf("unmarshal jsonData to pod failed: %v", err)
	}
}

func parseStrToPod(pod *v1.Pod) error {
	s, _, err := kyaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).
		Decode([]byte(podTmplate), nil, &unstructured.Unstructured{})
	if err != nil {
		klog.Errorf("kyaml NewDecodingSerializer failed: %v", err)
		return err
	}
	ss, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		klog.Errorf("kruntime DefaultUnstructuredConverter.ToUnstructured failed: %v", err)
		return err
	}
	if err = kruntime.DefaultUnstructuredConverter.FromUnstructured(ss, pod); err != nil {
		klog.Errorf("kruntime DefaultUnstructuredConverter.FromUnstructured failed: %v", err)
		return err
	}
	return nil
}

func main() {
	home := homeDir()
	if home == "" {
		klog.Fatalf("can't find the value ENV: HOME")
	}
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatalf("build kubeconfig failed: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("init clientset failed: %v", err)
	}

	flag.Parse()
	pod := &v1.Pod{}
	if *filePath != "" {
		parseFileToPod(*filePath, pod)
	} else {
		if err := parseStrToPod(pod); err != nil {
			klog.Fatalf("parse pod failed: %v", err)
		}
	}
	createPod(*podNumber, pod, clientset)
}
