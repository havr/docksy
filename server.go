package docksy

import (
	"crypto/tls"
	"strings"
	"net/http"
	"net"
	"sync"
	"fmt"
	"regexp"
	"net/url"
	"net/http/httputil"
)

type Server struct {
	httpListener net.Listener
	tlsListener net.Listener
	serving sync.WaitGroup

	redirect *RouteMap
	tlsHosts map[string] struct{}
}

func NewServer(listenHttp, listenHttps, etcd, certPath, docker, dockerCertPath, containerDirectory string) (server *Server, err error) {
	server = &Server{}

	if certPath != "" {
		var certs []tls.Certificate
		if certs, err = LoadCerts(certPath); err != nil {
			return
		}

		if server.tlsHosts, err = HostsFromCerts(certs); err != nil {
			return
		}

		for tlsHost, _ := range server.tlsHosts {
			fmt.Println("CERTS FOR", tlsHost)
		}

		tlsConfig := &tls.Config{Certificates: certs}

		var innerListener net.Listener
		if innerListener, err = net.Listen("tcp", listenHttps); err != nil {
			return
		}

		server.tlsListener = tls.NewListener(innerListener, tlsConfig)
	} else {
		server.tlsHosts = make(map[string] struct{})
	}

	server.redirect = NewRouteMap(etcd, docker, dockerCertPath, containerDirectory)

	if server.httpListener, err = net.Listen("tcp", listenHttp); err != nil {
		return
	}
	return
}

func (s *Server) Start() {
	go http.Serve(s.httpListener, s.wrapHandler(s.handleHTTP))
	if s.tlsListener != nil {
		go http.Serve(s.tlsListener, s.wrapHandler(s.handle))
	}
}

func (s *Server) Stop() {
	if s.tlsListener != nil {
		s.tlsListener.Close()
	}
	s.httpListener.Close()
	s.serving.Wait()
	s.redirect.Close()
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.URL
	host := s.stripPort(r.Host)
	if s.canServeUsingTls(host) {
		url.Scheme = "https"
		http.Redirect(w, r, "https://" + url.String(), http.StatusFound)
		return
	}

	s.handle(w, r)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	host := s.stripPort(r.Host)
	ip := s.redirect.Get(host)
	if ip == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	fmt.Println("REDIRECT", host, "->", host, ip)

	var u *url.URL
	var err error
	if u, err = url.Parse("http://" + ip); err != nil {
		return
	}

	rp := httputil.NewSingleHostReverseProxy(u)
	rp.ServeHTTP(w, r)
}

func (s *Server) stripPort(hostPort string) (host string) {
	if portIndex := strings.LastIndex(hostPort, ":"); portIndex != -1 {
		host = hostPort[:portIndex]
	} else {
		host = hostPort
	}
	return
}

func (s *Server) wrapHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.serving.Add(1)
		defer func() {
			if fail := recover(); fail != nil {
				http.Error(w, fmt.Sprintf("%v", fail), http.StatusInternalServerError)
			}
			s.serving.Done()
		} ()

		handler(w, r)
	}
}

func (s *Server) canServeUsingTls(host string) bool {
	for allowedHost, _ := range s.tlsHosts {
		if ok, err := regexp.MatchString(allowedHost, host); ok && err != nil {
			return true
		}
	}
	return false
}
