FROM golang

ADD . /go/src/github.com/havr/docksy
RUN go get github.com/fsouza/go-dockerclient
RUN go get github.com/coreos/go-etcd/etcd
RUN go install github.com/havr/docksy/main

CMD /go/bin/main

EXPOSE 80 443
