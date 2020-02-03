package goweb

import (
	"net/http"
	"regexp"
	"strings"

	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace/propagation"
	"golang.org/x/sync/errgroup"
	"github.com/sirupsen/logrus"
	"net"
	"fmt"
)

var Propagation propagation.HTTPFormat

func Run(servers ...Server) error {
	group := &errgroup.Group{}
	for _, server := range servers {
		srv := server
		entry := logrus.WithField("name", srv.Name())

		group.Go(func() error {
			entry.Debug("Listening...")
			return srv.Run()
		})

		OnShutdown(func() {
			entry.Debug("Shutting down...")
			if err := srv.Stop(); err != nil {
				entry.WithError(err).Warnf("Fail to shutdown: %s\n", srv.Name())
			} else {
				entry.Debugf("%s stopped\n", srv.Name())
			}
		})
	}
	return group.Wait()
}

func Grpc(name, addr string, server GrpcServer) Server {
	return &grpc{
		name: name,
		addr: addr,
		srv: server,
	}
}

func Http(name, addr string, handler http.Handler) Server {
	return &rest{
		name: name,
		srv: &http.Server{
			Addr: addr,
			Handler: &ochttp.Handler{
				Handler:          handler,
				Propagation:      Propagation,
				FormatSpanName:   newSpanNameFormatter(),
				IsHealthEndpoint: newHealthEndpoint(),
			},
		},
	}
}

func newSpanNameFormatter() func(*http.Request) string {
	const (
		repl        = "/-"
		defaultName = "/"
	)
	pattern := regexp.MustCompile(`/(\d+)`)
	return func(req *http.Request) string {
		if name := pattern.ReplaceAllString(req.URL.Path, repl); name != "" {
			return name
		}
		return defaultName
	}
}

func newHealthEndpoint() func(*http.Request) bool {
	endpoints := []string{"/metrics", "/debug/pprof"}
	return func(req *http.Request) bool {
		path := strings.TrimSuffix(req.URL.Path, "/")
		for _, endpoint := range endpoints {
			if strings.HasSuffix(path, endpoint) {
				return true
			}
		}
		return false
	}
}

type Server interface {
	Name() string
	Run() error
	Stop() error
}

type rest struct {
	name string
	srv *http.Server
}

func (r rest) Name() string {
	return fmt.Sprintf("Http (%s), %s", r.srv.Addr, r.name)
}

func (r rest) Run() error {
	return r.srv.ListenAndServe()
}

func (r rest) Stop() error {
	return r.srv.Shutdown(context.Background())
}

type GrpcServer interface {
	Serve(listener net.Listener) error
	Stop()
}

type grpc struct {
	name string
	addr string
	srv GrpcServer
}

func (g grpc) Name() string {
	return fmt.Sprintf("GRPC (%s), %s", g.addr, g.name)
}

func (g grpc) Run() error {
	if listener, err := net.Listen("tcp", g.addr); err != nil {
		return err
	} else {
		return g.srv.Serve(listener)
	}
}

func (g grpc) Stop() error {
	g.srv.Stop()
	return nil
}
