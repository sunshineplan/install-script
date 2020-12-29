package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
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

const api = "https://api.github.com/repos/v2fly/v2ray-core/releases/latest"

var client = http.DefaultClient

var downloadList = []string{"geoip.dat", "geosite.dat",
	"v2ctl.exe", "v2ray.exe", "wv2ray.exe"}

var tag, downloadURL, process string

func main() {
	getDownloadURL()
	if checkVersion() {
		fmt.Println("New version found.")
		fmt.Printf("Downloading %s\n", tag)
		b := download()
		checkProcess()
		replace(b)
		if process != "" {
			fmt.Println("Restarting V2Ray service.")
			if err := exec.Command(process).Start(); err != nil {
				log.Fatalln("Failed to restart v2ray:", err)
			}
		}
	} else {
		fmt.Printf("Latest version %s is already installed.\n", tag)
	}
	fmt.Println("Press enter key to continue . . .")
	fmt.Scanln()
}

func getDownloadURL() {
	var releases struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			DownloadURL string `json:"browser_download_url"`
		}
	}
	if err := gohttp.GetWithClient(api, nil, client).JSON(&releases); err != nil {
		log.Fatal(err)
	}
	for _, i := range releases.Assets {
		if strings.Contains(i.DownloadURL, "windows-64") && !strings.Contains(i.DownloadURL, "dgst") {
			tag = releases.TagName
			downloadURL = i.DownloadURL
			return
		}
	}
	log.Fatal("Not found")
}

func checkVersion() bool {
	command := exec.Command("v2ray", "-version")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	if err := command.Run(); err != nil {
		log.Fatalln("Failed to check v2ray version:", err)
	}
	return tag != "v"+strings.Split(stdout.String(), " ")[1]
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
	pb := progressbar.New(total).SetUnit("bytes")
	var out bytes.Buffer
	if _, err := pb.FromReader(resp.Body, &out); err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	<-pb.Done
	return out.Bytes()
}

func checkProcess() {
	processes, err := ps.Processes()
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range processes {
		if strings.Contains(p.Executable(), "v2ray") {
			fmt.Println("Shutting down V2Ray service.")
			process = p.Executable()
			if err := (&os.Process{Pid: p.Pid()}).Kill(); err != nil {
				log.Fatal(err)
			}
			time.Sleep(time.Second)
			break
		}
	}
}

func replace(b []byte) {
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range zr.File {
		for _, file := range downloadList {
			if i.FileHeader.Name == file {
				r, err := i.Open()
				if err != nil {
					log.Fatal(err)
				}
				f, err := os.Create(file)
				if err != nil {
					log.Fatal(err)
				}
				defer f.Close()
				if _, err := io.Copy(f, r); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
