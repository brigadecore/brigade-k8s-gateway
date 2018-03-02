package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/Azure/brigade/pkg/brigade"
	"github.com/Azure/brigade/pkg/storage"
	"github.com/Azure/brigade/pkg/storage/kube"
)

// This is adapted from the Brigade controller.

// init processes the flags.
func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	var (
		kubeconfig string
		master     string
		namespace  string
		configFile string
	)

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.StringVar(&namespace, "namespace", defaultNamespace(), "kubernetes namespace")
	flag.StringVar(&configFile, "config", defaultConfig(), "path to JSON configuration file. If no file is passed, you get to drink from the fire hose.")
	flag.Parse()

	var filters = &Config{}
	if configFile != "" {
		raw, err := ioutil.ReadFile(configFile)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(raw, filters); err != nil {
			panic(err)
		}
	}

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		log.Println("Missing -kubectl?")
		log.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new gateway listener.
	// TODO: We need a way of passing in appropriate filter information here.
	gw := newGateway(clientset, namespace)
	gw.config = filters
	gw.store = kube.New(clientset, namespace)
	log.Printf("Listening in namespace %q for new events", namespace)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go gw.Run(1, stop)

	// Wait forever
	select {}
}

type gateway struct {
	clientset kubernetes.Interface
	namespace string
	config    *Config
	store     storage.Store

	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	indexer  cache.Indexer
}

func newGateway(client kubernetes.Interface, ns string) *gateway {
	this := &gateway{
		clientset: client,
		namespace: ns,
		queue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	this.createInformer()
	return this
}

func (this *gateway) HasSynced() bool {
	return this.informer.HasSynced()
}

func (this *gateway) Run(numWorkers int, stopCh chan struct{}) {
	defer func() {
		this.queue.ShutDown()
		utilruntime.HandleCrash()
	}()

	log.Print("Watching Kubernetes for new events")

	go this.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, this.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < numWorkers; i++ {
		go wait.Until(this.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Print("Stopping Secret controller")
}

func (this *gateway) runWorker() {
	for this.processNextItem() {
	}
}

func (this *gateway) processNextItem() bool {
	key, done := this.queue.Get()
	if done {
		return false
	}
	defer this.queue.Done(key)

	err := this.sync(key.(string))
	this.handleErr(err, key)
	return true
}

func (this *gateway) createInformer() {
	sel := labels.Set{
	//"heritage": "helm",
	}

	// Initial test: Just watch all events.
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = sel.String()
			return this.clientset.CoreV1().Events(this.namespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = sel.String()
			return this.clientset.CoreV1().Events(this.namespace).Watch(options)
		},
	}
	reh := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				log.Println("Adding to workqueue: ", key)
				this.queue.Add(key)
			}
		},
	}

	this.indexer, this.informer = cache.NewIndexerInformer(lw, &v1.Event{}, 0, reh, cache.Indexers{})
}

func (this *gateway) sync(key string) error {
	raw, ok, err := this.indexer.GetByKey(key)
	if err != nil {
		return err
	}

	if !ok {
		fmt.Printf("Key %s is gone", key)
		return nil
	}

	e := raw.(*v1.Event)

	payload, _ := json.MarshalIndent(e, "", "  ")
	fmt.Printf("Processing %s: %s\n", key, string(payload))
	if !this.acceptEvent(e) {
		log.Printf("Rejecting %s by filter", key)
		return nil
	}
	return this.createSecret(e)
}

// filterEvent checks an event to see if it should be included or excluded
func (this *gateway) acceptEvent(e *v1.Event) bool {
	for i, f := range this.config.Filters {
		pass := f.Action == "accept"
		if f.Namespace != "" && e.InvolvedObject.Namespace != f.Namespace {
			log.Printf("rule %d: namespace mismatch", i)
			continue
		}
		if f.Kind != "" && e.InvolvedObject.Kind != f.Kind {
			log.Printf("rule %d: kind mismatch", i)
			continue
		}
		if len(f.Reasons) == 0 {
			log.Printf("rule %d: matched all conditions of rule", i)
			return pass
		}
		for ii, reason := range f.Reasons {
			if reason == e.Reason {
				log.Printf("rule %d: pass filter %d %q", i, ii, reason)
				return pass
			}
			log.Printf("rule %d: failed filter %d %q", i, ii, reason)
		}
	}
	// Default is reject
	log.Print("default rejection")
	return false
}

func (this *gateway) createSecret(e *v1.Event) error {
	name := fmt.Sprintf("%s:%s", e.InvolvedObject.Kind, e.Reason)
	// If the payload fails to marshal, we send along
	// without a payload.
	payload, _ := json.Marshal(e)
	b := &brigade.Build{
		ProjectID: this.config.Project,
		Type:      name,
		Provider:  "k8s-gateway",
		Revision:  &brigade.Revision{Ref: "refs/heads/master"},
		Payload:   payload,
	}

	// FIXME: This should be removed when the worker is fixed
	// Right now, the worker is not correctly falling back to the VCS to get
	// its brigade.js
	proj, err := this.store.GetProject(this.config.Project)
	if err != nil {
		return err
	}
	script, err := githubBrigadeJS(proj.Repo.Name, "master")
	if err != nil {
		return err
	}
	b.Script = script
	// END FIX

	j, _ := json.MarshalIndent(b, "", "  ")
	log.Printf("Build: %s\n", j)
	return this.store.CreateBuild(b)
}

func githubBrigadeJS(project, commit string) ([]byte, error) {
	// https://raw.githubusercontent.com/Azure/brigade/master/brigade.js
	project = strings.Replace(project, "github.com/", "", 1)
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/brigade.js", project, commit)
	log.Printf("Fetching %s", url)
	res, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}

	if res.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("could not get %s: %s", url, res.Status)
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

func (this *gateway) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		this.queue.Forget(key)
		return
	}

	if this.queue.NumRequeues(key) < 5 {
		log.Printf("retry for %s: %s", key, err)
		this.queue.AddRateLimited(key)
		return
	}

	this.queue.Forget(key)
	utilruntime.HandleError(err)
	log.Printf("Dropping %s from queue: %s", key, err)
}

func defaultNamespace() string {
	if ns, ok := os.LookupEnv("GATEWAY_NAMESPACE"); ok {
		return ns
	}
	return v1.NamespaceDefault
}

func defaultConfig() string {
	return os.Getenv("GATEWAY_CONFIG")
}

// Config represents a k8s-gateway configuration.
type Config struct {
	// Project is a project ID for a Brigade project.
	Project string `json:"project"`
	// Filters is a list of Filter objects that will be applied to incomming events.
	Filters []Filter `json:"filters"`
}

// Filter is a filter that will be applied against a Kubernets core Event object.
type Filter struct {
	// Namespace is a Kubernetes namespace
	// If set, only items in this namespace will match the filter. If
	// empty or *, all namespaces will match.
	Namespace string `json:"namespace"`
	// Kind is the Kubernetes Kind that this rule matches. Examples: Pod, Node
	Kind string `json:"kind"`
	// Reasons is a list of reasons strings to match. Evaluated first to last.
	// As soon as a reason matches, the Action will be taken
	// TODO: Should we support regular expressions here?
	Reasons []string `json:"reasons"`
	// Action to perform when filtering. One of "accept", "reject"
	// Unknown string will cause a rejection.
	Action string `json:"action"`
}
