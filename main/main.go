package main

import (
	"github.com/havr/docksy"
	"fmt"
	"os"
	"os/signal"
	"flag"
)

func run(listenHttp, listenHttps, etcd, certs, docker, dockerCerts, containerDirectory string) (err error) {
	var server *docksy.Server
	if server, err = docksy.NewServer(listenHttp, listenHttps, etcd, certs, docker, dockerCerts, containerDirectory); err != nil {
		return
	}

	server.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<- ch

	server.Stop()
	return
}

func main() {
	etcd := flag.String("etcd", "172.72.1.42", "etcd host")
	docker := flag.String("docker", "172.72.1.42", "docker host")
	dockerCerts := flag.String("docker-certs", "", "docker cert path")
	certs := flag.String("certs", "", "https certificate path")
	configDirectory := flag.String("config", "docksy", "etcd directory with config")
	listenHttp := flag.String("listen-http", ":80", "address http listener")
	listenHttps := flag.String("listen-https", ":443", "address https listener")
	flag.Parse()

	if err := run(*listenHttp, *listenHttps, *etcd, *certs, *docker, *dockerCerts, *configDirectory); err != nil {
		fmt.Println(err)
	}
}