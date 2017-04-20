package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/matrix-org/dendrite/common"
	"github.com/matrix-org/dendrite/syncserver/config"
	"github.com/matrix-org/dendrite/syncserver/consumers"
	"github.com/matrix-org/dendrite/syncserver/routing"
	"github.com/matrix-org/dendrite/syncserver/storage"
	"github.com/matrix-org/dendrite/syncserver/sync"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var configPath = flag.String("config", "sync-server-config.yaml", "The path to the config file. For more information, see the config file in this repository.")
var bindAddr = flag.String("listen", ":4200", "The port to listen on.")

func loadConfig(configPath string) (*config.Sync, error) {
	contents, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg config.Sync
	if err = yaml.Unmarshal(contents, &cfg); err != nil {
		return nil, err
	}
	// check required fields
	return &cfg, nil
}

func main() {
	common.SetupLogging(os.Getenv("LOG_DIR"))

	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config must be supplied")
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Invalid config file: %s", err)
	}

	if *bindAddr == "" {
		log.Fatal("--listen must be supplied")
	}

	log.Info("sync server config: ", cfg)

	db, err := storage.NewSyncServerDatabase(cfg.DataSource)
	if err != nil {
		log.Panicf("startup: failed to create sync server database with data source %s : %s", cfg.DataSource, err)
	}

	rp, err := sync.NewRequestPool(db)
	if err != nil {
		log.Panicf("startup: Failed to create request pool : %s", err)
	}

	server, err := consumers.NewServer(cfg, rp, db)
	if err != nil {
		log.Panicf("startup: failed to create sync server: %s", err)
	}
	if err = server.Start(); err != nil {
		log.Panicf("startup: failed to start sync server")
	}

	log.Info("Starting sync server on ", *bindAddr)
	routing.SetupSyncServerListeners(http.DefaultServeMux, http.DefaultClient, *cfg, rp)
	log.Fatal(http.ListenAndServe(*bindAddr, nil))
}