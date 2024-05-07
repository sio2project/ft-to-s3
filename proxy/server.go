package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/sio2project/ft-to-s3/v1/proxy/handlers"
	"github.com/sio2project/ft-to-s3/v1/utils"
)

func createHandlers(mux *http.ServeMux) {
	pathToHandler := map[string]func(http.ResponseWriter, *http.Request, utils.LoggerObject){
		"/version": handlers.Version,
	}

	for path, handler := range pathToHandler {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			instance := ctx.Value("instance").(utils.Instance)
			logger := utils.NewBucketLogger(instance.BucketName)

			w.Header().Set("Status-Code", "200")
			handler(w, r, logger)
			logger.Info("Request", r.URL.Path, "- status code", w.Header().Get("Status-Code"))
		})
	}
}

func Start(config *utils.Config) {
	mux := http.NewServeMux()
	createHandlers(mux)

	ctx, cancel := context.WithCancel(context.Background())
	servers := make([]*http.Server, 0, len(config.Instances))
	for _, inst := range config.Instances {
		server := &http.Server{
			Addr:    inst.Port,
			Handler: mux,
			BaseContext: func(listener net.Listener) context.Context {
				return context.WithValue(ctx, "instance", inst)
			},
		}
		servers = append(servers, server)
	}

	for _, server := range servers {
		go func(server *http.Server) {
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				utils.MainLogger.Error(fmt.Sprintf("Server on port %s failed to start: %+v", server.Addr, err))
			}
			cancel()
		}(server)
	}

	<-ctx.Done()
}
