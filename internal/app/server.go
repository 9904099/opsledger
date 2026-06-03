package app

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/9904099/opsledger/internal/discovery"
	"github.com/9904099/opsledger/internal/store"
)

type Server struct {
	store              store.Store
	awsImporter        *discovery.AWSImporter
	cloudflareImporter *discovery.CloudflareImporter
	pveImporter        *discovery.PVEImporter
	static             http.Handler
	index              []byte
	probeCancel        context.CancelFunc
	probeDone          chan struct{}
	syncCancel         context.CancelFunc
	syncDone           chan struct{}
	syncMu             sync.Mutex
	syncRunning        map[string]bool
}

type ServerConfig struct {
	Database store.DatabaseConfig
}

func NewServer(dataPath string) (*Server, error) {
	return NewServerWithConfig(ServerConfig{
		Database: store.DatabaseConfig{Driver: "sqlite", Path: dataPath},
	})
}

func NewServerWithConfig(config ServerConfig) (*Server, error) {
	dbStore, err := store.NewStore(config.Database)
	if err != nil {
		return nil, err
	}

	staticFS, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		_ = dbStore.Close()
		return nil, err
	}

	index, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		_ = dbStore.Close()
		return nil, err
	}

	server := &Server{
		store:              dbStore,
		awsImporter:        discovery.NewAWSImporter(dbStore),
		cloudflareImporter: discovery.NewCloudflareImporter(dbStore),
		pveImporter:        discovery.NewPVEImporter(dbStore),
		static:             http.FileServerFS(staticFS),
		index:              index,
		syncRunning:        map[string]bool{},
	}
	server.startAutoProbe()
	server.startAutoSync()
	return server, nil
}

func (s *Server) Close() error {
	if s.probeCancel != nil {
		s.probeCancel()
		select {
		case <-s.probeDone:
		case <-time.After(3 * time.Second):
			log.Printf("auto probe worker did not stop within timeout")
		}
	}
	if s.syncCancel != nil {
		s.syncCancel()
		select {
		case <-s.syncDone:
		case <-time.After(3 * time.Second):
			log.Printf("auto sync worker did not stop within timeout")
		}
	}
	return s.store.Close()
}
