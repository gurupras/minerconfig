package minerconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gurupras/go-stoppable-net-listener"
	"github.com/homesound/simple-websockets"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

type Pool struct {
	Url       string `json:"url"`
	User      string `json:"user"`
	Pass      string `json:"pass"`
	Keepalive bool   `json:"keepalive"`
	Nicehash  bool   `json:"nicehash"`
}

var pools []Pool

func RunServer(webserverPath string, port int) *stoppablenetlistener.StoppableNetListener {
	r := mux.NewRouter()
	ws := websockets.NewServer(r)
	ws.UseEvents = true

	ws.On("get-pools", func(w *websockets.WebsocketClient, data interface{}) {
		str, _ := json.Marshal(pools)
		w.Emit("get-pools", str)
	})

	ws.On("set-pools", func(w *websockets.WebsocketClient, data interface{}) {
		str := data.(string)
		var m map[string][]Pool
		if err := json.Unmarshal([]byte(str), &m); err != nil {
			log.Errorf("Failed to set-pools: %v", err)
			w.Emit("error", fmt.Sprintf("Failed to set-pools: %v", err))
		} else {
			pools = m["pools"]
			log.Infof("Successfully executed set-pools!\n")
		}
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
