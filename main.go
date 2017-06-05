package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

func printContent(arr []string) {
	glog.V(1).Infof(" there are %d items\n", len(arr))
	for _, pod := range arr {
		fmt.Println(pod)
	}
}

type AppConfig struct {
	masterUrl      string
	kubeConfigPath string
	nameSpace      string
}

func addCommandLine(config *AppConfig) {
	flag.StringVar(&config.masterUrl, "masterUrl", "", "kubernetes Master url")
	flag.StringVar(&config.kubeConfigPath, "kubeconfig", "", "the path of the kubernetes config")
	flag.StringVar(&config.nameSpace, "namespace", "", "namespace of the kubernetes")
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	flag.CommandLine.Parse([]string{})
}

func getPodListerWatcher(client *kubernetes.Clientset, namespace string) *cache.ListWatch {
	//selector := fields.SelectorFromSet(nil)
	selector := fields.Everything()
	//namespaceAll := ""
	listWatch := cache.NewListWatchFromClient(client.CoreV1Client.RESTClient(),
		"pods",
		namespace,
		selector)

	return listWatch
}

func testReflector(lw cache.ListerWatcher) {
	stopCh := make(chan struct{})
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	cycle := time.Millisecond * 0
	r := cache.NewReflector(lw, &v1.Pod{}, store, cycle)

	r.RunUntil(stopCh)

	for i := 1; i < 2; i++ {
		time.Sleep(30 * time.Second)
		printContent(store.ListKeys())
	}

	time.Sleep(10 * time.Second)
	printContent(store.ListKeys())
	close(stopCh)
}

func getObjInfo(obj interface{}) (string, interface{}) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		glog.Warning("OnSync not metav1.Object.")
		return "", obj
	}

	name := metaObj.GetNamespace() + "/" + metaObj.GetName()
	return name, obj
}

//Consumer functions
func OnSync(obj interface{}) {
	name, obj := getObjInfo(obj)

	if pod, ok := obj.(*v1.Pod); ok {
		name := pod.GetNamespace() + "/" + pod.GetName()
		glog.V(1).Infof("OnSync:%s %s", name, pod.Status.Phase)
		return
	}
	glog.Warningf("OnSync:%s not pod.", name)
}

func OnAdd(obj interface{}) {
	name, obj := getObjInfo(obj)

	if pod, ok := obj.(*v1.Pod); ok {
		name := pod.GetNamespace() + "/" + pod.GetName()
		glog.V(1).Infof("OnAdd:%s %s", name, pod.Status.Phase)
		return
	}
	glog.Warningf("OnAdd:%s not pod.", name)
}

func OnDelete(obj interface{}) {
	name, obj := getObjInfo(obj)

	if pod, ok := obj.(*v1.Pod); ok {
		name := pod.GetNamespace() + "/" + pod.GetName()
		glog.V(1).Infof("OnDelete:%s %s", name, pod.Status.Phase)
		return
	}

	glog.Warningf("OnDelete:%s not pod.", name)
}

func OnUpdate(oldObj, newObj interface{}) {
	name, obj := getObjInfo(oldObj)
	if pod, ok := obj.(*v1.Pod); ok {
		oldPod, _ := obj.(*v1.Pod)
		name := pod.GetNamespace() + "/" + pod.GetName()
		glog.V(1).Infof("OnUpdate:%s, %s --> %s", name, oldPod.Status.Phase, pod.Status.Phase)
		return
	}
	glog.Warningf("OnUpdate:%s not pod.", name)
}

func testRawController(lw cache.ListerWatcher) {
	clientState := cache.NewStore(cache.MetaNamespaceKeyFunc)
	fifo := cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, clientState)
	objType := &v1.Pod{}
	resyncPeriod := time.Second * 180

	cfg := &cache.Config{
		Queue:            fifo,
		ListerWatcher:    lw,
		ObjectType:       objType,
		FullResyncPeriod: resyncPeriod,
		RetryOnError:     false,

		Process: func(obj interface{}) error {
			for _, d := range obj.(cache.Deltas) {
				switch d.Type {
				case cache.Sync, cache.Added, cache.Updated:
					if d.Type == cache.Sync {
						OnSync(d.Object)
					}

					if old, exists, err := clientState.Get(d.Object); err == nil && exists {
						if err := clientState.Update(d.Object); err != nil {
							return err
						}
						OnUpdate(old, d.Object)
					} else {
						if err := clientState.Add(d.Object); err != nil {
							return err
						}
						OnAdd(d.Object)
					}
				case cache.Deleted:
					if err := clientState.Delete(d.Object); err != nil {
						return err
					}
					OnDelete(d.Object)
				default:
					glog.Warningf("Unknown event type:%v", d.Type)
				}
			}

			return nil
		},
	}

	stopCh := make(chan struct{})
	controller := cache.New(cfg)
	go controller.Run(stopCh)

	for i := 1; i < 10; i++ {
		time.Sleep(30 * time.Second)
		printContent(clientState.ListKeys())
	}

	time.Sleep(10 * time.Second)
	printContent(clientState.ListKeys())
	close(stopCh)
}

func testPodInformer(lw cache.ListerWatcher) {
	podEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    OnAdd,
		UpdateFunc: OnUpdate,
		DeleteFunc: OnDelete,
	}

	resyncPeriod := time.Second * 60
	clientState, podInformer := cache.NewInformer(lw,
		&v1.Pod{},
		resyncPeriod,
		podEventHandler)

	stopCh := make(chan struct{})
	go podInformer.Run(stopCh)

	for i := 1; i < 10; i++ {
		time.Sleep(30 * time.Second)
		printContent(clientState.ListKeys())
	}

	time.Sleep(10 * time.Second)
	printContent(clientState.ListKeys())
}

func main() {
	config := AppConfig{}

	addCommandLine(&config)
	glog.V(1).Infof("begin test")
	defer glog.V(1).Infof("end of test")

	client := getKubeClient(&config.masterUrl, &config.kubeConfigPath)
	if client == nil {
		fmt.Println("failed to get kubeclient")
		return
	}

	listWatch := getPodListerWatcher(client, config.nameSpace)
	//testReflector(listWatch)
	//testRawController(listWatch)
	testPodInformer(listWatch)
}
