package main

import (
	//"fmt"
	"net/http"
	//"net/url"
	"os"
	"io"
	"strconv"
	"path/filepath"
	"errors"
	//"strings"
	//"time"
)

const (
	PORT      = 9000
)

func uploaderCode() []byte {
	return []byte(`
<script>
	//uploader is coming
</script>
	`)
}

func serveDir(wr http.ResponseWriter, fd *os.File, path string) error {
	names, err := fd.Readdirnames(0)
	if err != nil {
		return err
	}
	
	if path[len(path)-1] == '/' {
		path = ""
	} else {
		path = filepath.Base(path) + "/"
	}
	
	wr.Header().Set("Content-type", "text/html")
	wr.Write(uploaderCode())
	wr.Write([]byte("<a href=\""+path+"..\">..</a><br>"))
	for _, name := range names {
		wr.Write([]byte("<a href=\""+path+name+"\">"+name+"</a><br>"))
	}
	
	return nil
}

func serveFile(wr http.ResponseWriter, fd *os.File, path string) error {
	name := filepath.Base(path)
	wr.Header().Set("Content-type", "application/octet-stream")
	wr.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")

	_, err := io.Copy(wr, fd)
	if err != nil {
		return err
	}

	return nil
}

func servePath(wr http.ResponseWriter, path string) error {
	fd, err := os.Open(path);
	if err != nil {
		return err
	}
	defer fd.Close()
	
	fi, err := fd.Stat()
	if err != nil {
		return err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return serveDir(wr, fd, path)
	case mode.IsRegular():
		return serveFile(wr, fd, path)
	}
	
	return errors.New("Neither file nor dir");
}

func handler(wr http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		err := servePath(wr, "."+req.URL.Path)
		if err != nil {
			wr.Header().Set("Content-type", "text/plain")
			wr.Write([]byte(err.Error()))
		} else {
		
		}
	case "POST":
		wr.Header().Set("Content-type", "text/plain")
		wr.Write([]byte("POST"))
	}
}

func main() {
	http.HandleFunc("/", handler)

	listenAddr := ":" + strconv.Itoa(PORT)
	println("Starting on " + listenAddr)
	err := http.ListenAndServe(listenAddr, nil)
	panic(err)
}
