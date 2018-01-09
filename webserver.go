package minerconfig

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/gorilla/mux"
	"github.com/gurupras/go-easyfiles"
	"github.com/gurupras/go-stoppable-net-listener"
	"github.com/homesound/simple-websockets"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

var pools []interface{}
var selectedPools []interface{}

func RunServer(webserverPath string, port int) *stoppablenetlistener.StoppableNetListener {
	r := mux.NewRouter()
	ws := websockets.NewServer(r)
	ws.UseEvents = true

	pools = make([]interface{}, 0)

	poolsDir := filepath.Join(webserverPath, "pools")
	if !easyfiles.Exists(poolsDir) {
		easyfiles.Makedirs(poolsDir)
	}

	files, err := doublestar.Glob(filepath.Join(poolsDir, "pool-*"))
	if err != nil {
		log.Errorf("Failed to list pools in poolsDir: %v", err)
	}
	log.Debugf("Found %d pool files", len(files))
	for _, file := range files {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			log.Errorf("Failed to read file '%v': %v", file, err)
			continue
		}
		var pool map[string]interface{}
		if err = json.Unmarshal(b, &pool); err != nil {
			log.Errorf("Failed to unmarshal pool from file '%v': %v", file, err)
		} else {
			pools = append(pools, pool)
		}
	}

	// Load selected pools
	selectedPoolsPath := filepath.Join(poolsDir, "selected-pools")
	noSelectedPools := true
	if easyfiles.Exists(selectedPoolsPath) {
		noSelectedPools = false
		b, err := ioutil.ReadFile(selectedPoolsPath)
		if err != nil {
			log.Errorf("Failed to read selected pools file '%v': %v", selectedPoolsPath, err)
			noSelectedPools = true
		} else {
			if err = json.Unmarshal(b, &selectedPools); err != nil {
				log.Errorf("Failed to unmarshal selected pools file '%v': %v", selectedPoolsPath, err)
				noSelectedPools = true
			}
		}
	}
	if noSelectedPools {
		selectedPools = make([]interface{}, 0)
	}

	ws.On("update-selected-pools", func(w *websockets.WebsocketClient, data interface{}) {
		if ws.UseEvents {
			evt := &websockets.Event{"update-selected-pools", fmt.Sprintf("clientaddr=%v pools=%v", w.RemoteAddr(), data)}
			ws.EventChan <- evt
		}
		poolStr := data.(string)
		poolBytes := []byte(poolStr)
		if err := json.Unmarshal(poolBytes, &selectedPools); err != nil {
			log.Errorf("[update-selected-pools]: Failed to unmarshal: %v", err)
			return
		}
		for client, _ := range ws.Clients {
			client.Emit("update-selected-pools", selectedPools)
		}
	})

	ws.On("add-pool", func(w *websockets.WebsocketClient, data interface{}) {
		if ws.UseEvents {
			evt := &websockets.Event{"add-pool", fmt.Sprintf("clientaddr=%v pool=%v", w.RemoteAddr(), data)}
			ws.EventChan <- evt
		}
		poolStr := data.(string)
		poolBytes := []byte(poolStr)
		var pool map[string]interface{}
		if err := json.Unmarshal(poolBytes, &pool); err != nil {
			log.Errorf("[add-pool]: Failed to unmarshal pool: %v", err)
		}
		hash := fmt.Sprintf("%X", md5.Sum(poolBytes))
		poolFile := filepath.Join(poolsDir, fmt.Sprintf("pool-%v", hash))
		if !easyfiles.Exists(poolFile) {
			if err := ioutil.WriteFile(poolFile, poolBytes, 0666); err != nil {
				log.Errorf("Failed to write new pool to file: %v", err)
				w.Emit("error", fmt.Sprintf("Failed to write new pool to file: %v", err))
			}
		}
		pools = append(pools, pool)
		for client, _ := range ws.Clients {
			client.Emit("new-pool", pool)
		}
	})

	ws.On("get-available-pools", func(w *websockets.WebsocketClient, data interface{}) {
		if ws.UseEvents {
			evt := &websockets.Event{"get-available-pools", fmt.Sprintf("clientaddr=%v", w.RemoteAddr())}
			ws.EventChan <- evt
		}
		b, _ := json.Marshal(pools)
		_ = b
		w.Emit("get-available-pools-result", pools)
	})

	ws.On("get-selected-pools", func(w *websockets.WebsocketClient, data interface{}) {
		if ws.UseEvents {
			evt := &websockets.Event{"get-selected-pools", fmt.Sprintf("clientaddr=%v", w.RemoteAddr())}
			ws.EventChan <- evt
		}
		b, _ := json.Marshal(selectedPools)
		_ = b
		w.Emit("get-selected-pools-result", selectedPools)
	})

	webserverBasePath := webserverPath
	staticPath := filepath.Join(webserverBasePath, "static") + "/"

	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		indexFile := filepath.Join(staticPath, "html", "index.html")
		f, err := os.Open(indexFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Got error:", err)
		}
		defer f.Close()
		if _, err = io.Copy(w, f); err != nil {
			fmt.Fprintln(os.Stderr, "Got error:", err)
		}

	})
	r.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir(staticPath))))

	mux := http.NewServeMux()
	mux.Handle("/", r)
	corsHandler := cors.Default().Handler(mux)
	server := http.Server{}
	server.Handler = corsHandler
	snl, err := stoppablenetlistener.New(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get snl: %v\n", err)
		os.Exit(-1)
	}
	go func() {
		for evt := range ws.EventChan {
			log.Infof("Websocket event: %v", evt)
		}
	}()
	go func() {
		server.Serve(snl)
	}()
	return snl
}
