package main

import (
	//"fmt"
	"net/http"
	//"net/url"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	//"mime/multipart"
	"strings"
	//"time"
)

const (
	PORT                  = 9000
	MULTIPART_BUFFER_SIZE = 1024*1024
)

func uploaderCode(uploadDir string) []byte {
	return []byte(`
	<style>
		input {
			display: block;
		}
		.file-block {
			outline: 1px solid;
		}
	</style>
	<div id="fileBlockTemplate" style="display:none;" class="file-block">
		<span class="file-name"></span>
		<span class="transfer-progress"></span>
	</div>
	<form id='theForm' action='` + uploadDir + `' method='POST' enctype='multipart/form-data'>
		<input id="fileInput" type='file' name='file'>
		<input type='submit' value='Send'>
	</form>
	<span>Or just drag file to page</span>
	<div id="fileList"></div>
	<br>
	<br>
	<script>
		Number.prototype.sizefy = function() {
			if (this < 5*1024) return this+" B";
			if (this < 5*1024*1024) return ((this/1024)|0)+" KiB";
			if (this < 5*1024*1024*1024) return ((this/1024/1024)|0)+" MiB";
			return ((this/1024/1024/1024)|0)+" GiB";
		}
		function fileInputOnchange() {
			if (this.files.length == 0) return;
			addFiles(this.files);
			resetFileInput();
		}
		function resetFileInput() {
			fileInput.outerHTML = fileInput.outerHTML;
			fileInput.onchange = fileInputOnchange;
		}
		resetFileInput();
		function sendFiles(files, onProgress, onEnd) {
			var form = new FormData();
			for (var i=0; i<files.length; i++) {
				form.append('file'+i, files[i]);
			}
			
			var xhr = new XMLHttpRequest();
			xhr.upload.addEventListener('progress',onProgress, false);
			xhr.open(theForm.method, theForm.action, true);
			xhr.onreadystatechange = function() {
				if (xhr.readyState != 4) return;
				onEnd(xhr.status, xhr.responseText);
			}
			xhr.send(form);
		}
		function addFiles(files) {
			var block = fileBlockTemplate.cloneNode(true);
			block.id = block.style.display = null;
			
			var name = block.getElementsByClassName("file-name")[0];
			var names = new Array(files.length);
			for (var i=0; i<files.length; i++) names[i] = files[i].name;
			name.textContent = names.join(", ");
			
			var progressBox = block.getElementsByClassName("transfer-progress")[0];
			
			//block.filesToUpload = files;
			fileList.appendChild(block);
			var stt = new Date().getTime();
			sendFiles(files, function(e) {
				var ct = new Date().getTime();
				var percent = (e.loaded / e.total * 100)|0;
				var speed = (e.loaded / (ct-stt) * 1000)|0;
				progressBox.textContent = percent+"% "+speed.sizefy()+"/s";
			}, function(status, resText) {
				progressBox.textContent = resText;
				if (status == 200) setTimeout(function(){ fileList.removeChild(block) }, 3000);
			});
		}
		var elem = document.body;
		elem.ondragover = function() {
			this.style.outline = "5px solid blue";
			return false;
		}
		elem.ondragleave = function() {
			this.style.outline = null;
			return false;
		}
		elem.ondrop = function(e) {
			e.preventDefault();
			this.style.outline = null;
			addFiles(e.dataTransfer.files);
//			var length = e.dataTransfer.items.length;
//			for (var i = 0; i < length; i++) {
//				var entry = e.dataTransfer.items[i].webkitGetAsEntry();
//				if (entry.isFile) {
//					// do whatever you want
//				} else if (entry.isDirectory) {
//					// do whatever you want
//				}
//			}
		}
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
	wr.Write([]byte("<html>\n\t<head>\n\t\t<title>HFS</title>\n\t</head>\n<body>\n"))
	wr.Write(uploaderCode(path))
	wr.Write([]byte("<a href=\"" + path + "..\">..</a><br>\n"))
	for _, name := range names {
		wr.Write([]byte("<a href=\"" + path + name + "\">" + name + "</a><br>\n"))
	}
	wr.Write([]byte("</body>\n</html>\n"))

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
	fd, err := os.Open("." + path)
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

	return errors.New("Neither file nor dir")
}

func serveUpload(wr http.ResponseWriter, req *http.Request) error {
	reader, err := req.MultipartReader()
	if err != nil {
		return errors.New("While making reader: " + err.Error())
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.New("While reading part: " + err.Error())
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
			return errors.New("While creating: " + err.Error())
		}

		_, err = io.Copy(fd, part)
		if err != nil {
			return errors.New("While saving <" + name + ">: " + err.Error())
		}
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
