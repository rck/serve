package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

func main() {
	flagH := flag.String("host", "", "By default the FQDN is used, if set, overwrite with `string`")
	flagO := flag.Bool("o", false, "Serve file only once and then exit")
	flagP := flag.Int("p", 12345, "Port `number` used to share file. This and the next 10 ports will be tried")
	flagX := flag.Bool("x", false, "Copy url to clipboard (requires xclip)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s [OPTION]... [FILE|DIR]:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	fileArg := flag.Args()
	if len(fileArg) != 1 {
		flag.Usage()
	}

	var hostName string
	if *flagH != "" {
		hostName = *flagH
	} else {
		if h, err := fqdn(); err != nil {
			log.Fatal(err)
		} else {
			hostName = h
		}
	}

	file := fileArg[0]
	info, err := os.Stat(file)
	if err != nil {
		log.Fatal(err)
	}
	isDir := info.IsDir()

	fileBase := "/"
	if !isDir {
		fileBase += path.Base(file)
		http.HandleFunc(fileBase, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, file)
			if *flagO {
				os.Exit(0)
			}
			fmt.Println("Done")
		})
	}

	u, err := url.Parse(fileBase)
	if err != nil {
		log.Fatal(err)
	}

	for port := *flagP; port <= *flagP+10; port++ {
		url := fmt.Sprintf("http://%s:%d%s", hostName, port, u)
		fmt.Println("Serving", file, "as", url)
		if *flagX {
			if err := xclip(url); err != nil {
				log.Println(err)
			}
		}
		pStr := strconv.Itoa(port)
		if isDir {
			log.Println(http.ListenAndServe(":"+pStr, http.FileServer(http.Dir(file))))
		} else {
			log.Println(http.ListenAndServe(":"+pStr, nil))
		}
	}
}

func xclip(url string) error {
	cmd := exec.Command("xclip", "-i")
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write([]byte(url)); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return cmd.Wait()
}

func fqdn() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname, nil
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return hostname, nil
			}
			hosts, err := net.LookupAddr(string(ip))
			if err != nil || len(hosts) == 0 {
				return hostname, nil
			}
			fqdn := hosts[0]
			return strings.TrimSuffix(fqdn, "."), nil
		}
	}
	return hostname, nil
}
