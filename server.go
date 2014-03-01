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
	//"mime/multipart"
	"strings"
	//"time"
)

const (
	PORT      = 9000
	MULTIPART_BUFFER_SIZE = 1024
)

func uploaderCode(uploadDir string) []byte {
	return []byte(`
<form action='`+uploadDir+`' method='POST' enctype='multipart/form-data'>
	<input type='file' name='file'>
	<input type='submit' value='Send'>
</form>
<script>
	//uploader is coming
</script>`)
}

func serveDir(wr http.ResponseWriter, fd *os.File, path string) error {
	names, err := fd.Readdirnames(0)
	if err != nil {
		return err
	}
	
	// "/bla/bla/" + "filename" -> "/bla/bla/filename"
	// "/bla/bla" + "bla/filename" -> "/bla/bla/filename"
	if path[len(path)-1] == '/' {
		path = ""
	} else {
		path = filepath.Base(path) + "/"
	}
	
	wr.Header().Set("Content-type", "text/html")
	wr.Write(uploaderCode(path))
	wr.Write([]byte("<a href=\""+path+"..\">..</a><br>\n"))
	for _, name := range names {
		wr.Write([]byte("<a href=\""+path+name+"\">"+name+"</a><br>\n"))
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
	fd, err := os.Open("."+path);
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

func serveUpload(wr http.ResponseWriter, req *http.Request) error {
	reader, err := req.MultipartReader()
	if err != nil {
		return err
	}
	
	part, err := reader.NextPart()
	if err != nil {
		return err
	}
	
	name := part.FileName()
	if name == "" {
		return errors.New("Empty file name")
	}
	
	//buf := make([]byte, MULTIPART_BUFFER_SIZE, MULTIPART_BUFFER_SIZE)
	//n, err := part.Read(buf) //io.EOF
	
	path := "." + req.URL.Path
	if path[len(path)-1] != '/' {
		path += "/"
	}
	
	fd, err := os.Create(path + strings.Replace(name, "/", "_", -1))
	if err != nil {
		return err
	}
	
	_, err = io.Copy(fd, part)
	if err != nil {
		return err
	}
	
	wr.Header().Set("Content-type", "text/plain")
	wr.Write([]byte("OK"))
	
	return nil
}

func handler(wr http.ResponseWriter, req *http.Request) {
	var err error
	switch req.Method {
	case "GET":
		err = servePath(wr, req.URL.Path)
	case "POST":
		err = serveUpload(wr, req)
	default:
		err = errors.New("Unknown method")
	}
	if err != nil {
		wr.Header().Set("Content-type", "text/plain")
		wr.Write([]byte(err.Error()))
	}
}

func main() {
	http.HandleFunc("/", handler)

	listenAddr := ":" + strconv.Itoa(PORT)
	println("Starting on " + listenAddr)
	err := http.ListenAndServe(listenAddr, nil)
	panic(err)
}
