package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	ps "github.com/mitchellh/go-ps"
	"github.com/sunshineplan/gohttp"
	"github.com/sunshineplan/utils/progressbar"
)

const api = "https://api.github.com/repos/geph-official/gephng-binaries/commits"

var client = http.DefaultClient

var tag, downloadURL string
var process bool

var pb *progressbar.ProgressBar

type writeCounter struct{}

func (wc *writeCounter) Write(p []byte) (int, error) {
	pb.Add(len(p))
	return 0, nil
}

func main() {
	getDownloadURL()
	if checkVersion() {
		fmt.Printf("Downloading v%s\n", tag)
		b := download()
		checkProcess()
		replace(b)
		version, err := os.Create("version")
		if err != nil {
			log.Fatal(err)
		}
		defer version.Close()
		if _, err := version.WriteString(tag); err != nil {
			log.Fatal(err)
		}
		if process {
			fmt.Println("Restarting Geph service.")
			if err := exec.Command("python", "-c",
				"import subprocess;subprocess.Popen('geph -config client.conf',creationflags=134217728)").Run(); err != nil {
				log.Fatalln("Failed to restart geph:", err)
			}
		}
	}
	fmt.Println("Press enter key to continue . . .")
	fmt.Scanln()
}

func getDownloadURL() {
	var commits []struct {
		SHA string
	}
	if err := gohttp.GetWithClient(api, nil, client).JSON(&commits); err != nil {
		log.Fatal(err)
	}
	var commit struct {
		Files []struct {
			RawURL string `json:"raw_url"`
		}
	}
	if err := gohttp.GetWithClient(api+"/"+commits[0].SHA, nil, client).JSON(&commit); err != nil {
		log.Fatal(err)
	}
	for _, i := range commit.Files {
		if strings.Contains(i.RawURL, "geph-client-windows") {
			tag = strings.ReplaceAll(strings.Split(i.RawURL, "-v")[1], ".exe", "")
			downloadURL = i.RawURL
			return
		}
	}
	log.Fatal("Not found")
}

func checkVersion() bool {
	current, err := ioutil.ReadFile("version")
	if err != nil {
		fmt.Println("Check version failed.")
		return true
	}
	result := tag != strings.TrimSpace(string(current))
	if result {
		fmt.Println("New version found.")
	} else {
		fmt.Printf("Latest version v%s is already installed.\n", tag)
	}
	return result
}

func download() []byte {
	resp := gohttp.GetWithClient(downloadURL, nil, client)
	if resp.Error != nil {
		log.Fatal(resp.Error)
	}
	total, err := strconv.Atoi(resp.Header.Get("content-length"))
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan bool, 1)
	pb = progressbar.New(total, c)
	pb.Start()
	var out bytes.Buffer
	if _, err := io.Copy(&out, io.TeeReader(resp.Body, &writeCounter{})); err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	<-c
	return out.Bytes()
}

func checkProcess() {
	processes, err := ps.Processes()
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range processes {
		if strings.Contains(p.Executable(), "geph") {
			fmt.Println("Shutting down Geph service.")
			process = true
			if err := (&os.Process{Pid: p.Pid()}).Kill(); err != nil {
				log.Fatal(err)
			}
			time.Sleep(time.Second)
			break
		}
	}
}

func replace(b []byte) {
	f, err := os.Create("geph.exe")
	if err != nil {
		f, err = os.Create("geph.tmp")
		if err != nil {
			log.Fatal(err)
		} else {
			defer log.Fatal("Failed to replace geph.exe file.")
		}
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		log.Fatal(err)
	}
}
