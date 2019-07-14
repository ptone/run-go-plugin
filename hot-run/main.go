package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var restartChan chan error

func init() {
	restartChan = make(chan error, 1)
}

func main() {
	url, _ := url.Parse("http://localhost:6060")
	proxy := httputil.NewSingleHostReverseProxy(url)

	go func() {
		runServer()
	}()
	http.HandleFunc("/", proxy.ServeHTTP)
	http.HandleFunc("/_restart", reloader)
	http.HandleFunc("/_upload", uploadHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func startServer() *exec.Cmd {
	log.Print("starting sub-server")
	var err error
	if _, err = os.Stat("/tmp/new-app"); err == nil {
		log.Print("New Binary")
		if err = os.Remove("/tmp/app"); err != nil {
			log.Fatalf("Unable to remove app: %s", err)
		}
		if err = os.Rename("/tmp/new-app", "/tmp/app"); err != nil {
			log.Fatal(err)
		}
		if err = os.Chmod("/tmp/app", 0755); err != nil {
			log.Fatal(err)
		}
	}

	files, err := ioutil.ReadDir("/tmp/")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
		fmt.Println(f.Size())
		fmt.Println(f.ModTime())
	}

	cmd := exec.Command("/tmp/app")
	cmd.Env = append(os.Environ(),
		"PORT=6060",
	)
	// cmd.Dir = "/"
	if err := cmd.Start(); err != nil {
		log.Print("unable to start sub-server")
		log.Print(err.Error())
	} else {
		go func() {
			err := cmd.Wait()
			errmsg := err.Error()
			if strings.Contains(errmsg, "exited") || strings.Contains(errmsg, "killed") {
				log.Printf("caught exit: %s", errmsg)
			} else {
				log.Print(err.Error())
				restartChan <- errors.New("Crash?")
			}
			log.Print("sub server exited")
		}()
	}
	return cmd
}
func runServer() {
	var cmd *exec.Cmd
	for {
		cmd = startServer()
		err := <-restartChan
		log.Print(err.Error())
		if strings.Contains(err.Error(), "killed") {
			// not a fatal error if the process was explicitly killed
			continue
		}
		if strings.Contains(err.Error(), "reload") && cmd.ProcessState == nil {
			log.Print("killing process")
			cmd.Process.Kill()
		}

	}
}

func reloader(w http.ResponseWriter, r *http.Request) {
	restartChan <- errors.New("reload trigger")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Create("/tmp/new-app")
	if err != nil {
		panic(err)
	}
	n, err := io.Copy(file, r.Body)
	if err != nil {
		panic(err)
	}
	file.Sync()
	file.Close()
	restartChan <- errors.New("reload trigger")
	w.Write([]byte(fmt.Sprintf("%d bytes received.\n", n)))
	return
}
