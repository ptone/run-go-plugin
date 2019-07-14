// Copyright 2019 Google Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"time"

	"cloud.google.com/go/storage"
	"github.com/go-chi/chi"
)

// global variables
var handlerFunc func(http.ResponseWriter, *http.Request)
var storageClient *storage.Client
var ctx context.Context
var restartChan chan struct{}
var shutdownChan chan struct{}

func init() {
	ctx = context.Background()
	restartChan = make(chan struct{})
	shutdownChan = make(chan struct{})
	var err error
	// Creates a client.
	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	err = loadPlugin()
	if err != nil {
		handlerFunc = errorHandler
	}
}

func main() {
	for {
		startServer()
	}
}

func loadPlugin() error {
	bucket := os.Getenv("PLUGIN_BUCKET")
	// TODO consider taking obj name or URL as queryparam from update script
	object := "plugin.so"

	rc, err := storageClient.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return err
	}

	defer rc.Close()
	dir, err := ioutil.TempDir(os.TempDir(), "plugin-")
	if err != nil {
		return err
	}
	pluginPath := filepath.Join(dir, "plugin.so")
	// TODO use in memory file like object?
	w, err := os.Create(pluginPath)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, rc)
	if err != nil {
		return err
	}
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return err
	}

	h, err := p.Lookup("Handler")
	if err != nil {
		log.Printf("could not find Handler function: %v", err)
		return err
	}
	var ok bool
	handlerFunc, ok = h.(func(http.ResponseWriter, *http.Request))
	if !ok {
		log.Printf("found handler but type is %T instead of handlerFunc error\n", h)
		return err
	}
	return nil
}

func startServer() {
	log.Print("Starting Server")
	r := chi.NewRouter()
	r.Get("/", handlerFunc)
	r.Get("/_reload", reloader)
	r.Get("/_die", killme)
	server := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			// the server stopping is expected and not an error we want to die on
			if err.Error() != "http: Server closed" {
				log.Fatal(err)
			}
		}
	}()

	select {
	case <-restartChan:
		log.Print("graceful restart")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			panic("Unable to shutdown server")
		}
	case <-shutdownChan:
		log.Print("starting shutdown")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			panic("Unable to shutdown server")
		}
		log.Fatal("bye")
	}
	return
}

func reloader(w http.ResponseWriter, r *http.Request) {
	log.Print("reloading...")
	err := loadPlugin()
	if err != nil {
		handlerFunc = errorHandler
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "no plugin found")
	} else {
		fmt.Fprint(w, "ok")
	}
	log.Print("done")
	restartChan <- struct{}{}
	return
}

func killme(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "reseting")
	shutdownChan <- struct{}{}
	return
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("No Plugin Found\n"))
	return
}
