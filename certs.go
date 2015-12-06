package docksy

import (
	"crypto/tls"
	"strings"
	"os"
	"io/ioutil"
	"crypto/x509"
)

// LoadCerts loads all certificates from a given directory
func LoadCerts(path string) (certs []tls.Certificate, err error) {
	path = strings.TrimSuffix(path, "/")
	certs = make([]tls.Certificate, 0)

	var dir []os.FileInfo
	if dir, err = ioutil.ReadDir(path); err != nil {
		return
	}

	processed := make(map[string] struct{})
	for _, file := range dir {
		name := file.Name()
		if file.IsDir() {
			var innerCerts []tls.Certificate
			if innerCerts, err = LoadCerts(path + "/" + name); err != nil {
				err = nil
				continue
			}

			certs = append(certs, innerCerts...)
			continue
		}

		crtSuffix := ".crt"
		keySuffix := ".key"
		if !(strings.HasSuffix(name, crtSuffix) || strings.HasSuffix(name, keySuffix)) {
			continue
		}

		name = strings.TrimSuffix(name, crtSuffix)
		name = strings.TrimSuffix(name, keySuffix)
		if _, ok := processed[name]; ok {
			continue
		}

		crtFile := path + "/" + name + crtSuffix
		keyFile := path + "/" + name + keySuffix

		var cert tls.Certificate
		if cert, err = tls.LoadX509KeyPair(crtFile, keyFile); err != nil {
			err = nil
			continue
		}

		processed[name] = struct{}{}
		certs = append(certs, cert)
	}

	return
}

// HostsFromCerts gets hosts for which certificate is valid for each of given certs
// and forms a set from them
func HostsFromCerts(certs []tls.Certificate) (hosts map[string] struct{}, err error) {
	hosts = make(map[string] struct{})
	for _, cert := range certs {
		for _, part := range cert.Certificate {
			var parsed *x509.Certificate
			if parsed, err = x509.ParseCertificate(part); err != nil {
				break
			}
			for _, domain := range parsed.DNSNames {
				if domain == "" {
					continue
				}
				hosts[domain] = struct{}{}
			}
		}
	}
	return
}
