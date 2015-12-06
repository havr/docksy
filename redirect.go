package docksy

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/coreos/go-etcd/etcd"
	"sync"
	"fmt"
	"os"
	"strings"
	"time"
)

type RouteMap struct {
	client *etcd.Client
	docker *docker.Client
	stop chan bool
	hosts map[string] string
	running sync.WaitGroup
	lock sync.RWMutex
	configDirectory string
}

// NewRouteMap starts a service that fetches routes from configDirectory of etcdAddress
// and translates container name to its internal ip for each route entry
func NewRouteMap(etcdAddress string, dockerEndpoint, dockerCertPath, configDirectory string) (rm *RouteMap) {
	rm = &RouteMap{}
	var err error
	if dockerCertPath == "" {
		if rm.docker, err = docker.NewClient(dockerEndpoint); err != nil {
			panic(err)
		}
	} else {
		ca := fmt.Sprintf("%s/ca.pem", dockerCertPath)
		cert := fmt.Sprintf("%s/cert.pem", dockerCertPath)
		key := fmt.Sprintf("%s/key.pem", dockerCertPath)

		if rm.docker, err = docker.NewTLSClient(dockerEndpoint, cert, key, ca); err != nil {
			panic(err)
		}
	}

	if rm.client = etcd.NewClient([]string{etcdAddress}); err != nil {
		panic(err)
	}

	if !strings.HasPrefix(configDirectory, "/") {
		configDirectory = "/" + configDirectory
	}
	if !strings.HasSuffix(configDirectory, "/") {
		configDirectory = configDirectory + "/"
	}
	rm.configDirectory = configDirectory

	rm.stop = make(chan bool, 1)
	rm.hosts = make(map[string] string)
	go rm.watch()
	return
}

// update fetches configuration for all routes from etcd and sets route map
func (rm *RouteMap) update() bool {
	var err error
	var response *etcd.Response
	if response, err = rm.client.Get(rm.configDirectory, false, false); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	for _, node := range response.Node.Nodes {
		rm.setNode(node)
	}
	return true
}

// hostFromNode returns host name from a given etcd node
func (rm *RouteMap) hostFromNode(node *etcd.Node) string {
	return strings.TrimPrefix(node.Key, rm.configDirectory)
}

// setNode sets route map according to a given etcd entry
func (rm *RouteMap) setNode(node *etcd.Node) {
	var err error
	var container *docker.Container
	if container, err = rm.docker.InspectContainer(node.Value); err != nil {
		return
	}
	ip := container.NetworkSettings.IPAddress

	rm.lock.Lock()
	defer rm.lock.Unlock()

	host := rm.hostFromNode(node)
	rm.hosts[host] = ip
	fmt.Println("SET ROUTE", host, "->", node.Value, ip)
}

// watch listens etcd directory for changes
func (rm *RouteMap) watch() {
	rm.running.Add(1)
	defer rm.running.Done()

	response := make(chan *etcd.Response, 1)
	defer close(response)

	stop := make(chan bool, 1)
	for {
		if !rm.update() {
			time.Sleep(1 * time.Second)
			continue
		}

		go rm.client.Watch(rm.configDirectory, 0, true, response, stop)
		for {
			select {
			case <- rm.stop:
				close(stop)
				return
			case resp := <- response:
				if resp == nil {
					break
				}
				rm.handleUpdateResponse(resp)
			}
		}
	}
}

// handleUpdateResponse updates route map according to etcd change entry
func (rm *RouteMap) handleUpdateResponse(resp *etcd.Response) {
	node := resp.Node
	if resp.Action == "delete" {
		rm.deleteNode(node)
	} else {
		rm.setNode(node)
	}
}

// deleteNode deletes entry from the route map
func (rm *RouteMap) deleteNode(node *etcd.Node) {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	name := rm.hostFromNode(node)
	fmt.Println("DELETE ROUTE", name, "->", rm.hosts[name])
	delete(rm.hosts, name)
}

// get returns ip to route request for given host
func (rm *RouteMap) Get(host string) (ip string) {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	ip, _ = rm.hosts[host]
	return
}

// close shuts down route map
func (rm *RouteMap) Close() {
	close(rm.stop)
	rm.running.Wait()
	rm.client.Close()
}
