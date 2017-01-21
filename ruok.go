package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/parallellink/srg"
)

var (
	defaultPorts   = []string{"11379", "11380", "11381"}
	timeout        = 1 * time.Second
	sectionPattern = regexp.MustCompile(`^# (.+)$`)
)

func main() {

	hostsExp := flag.String("h", "", "hosts expression")
	portsExp := flag.String("p", "", "ports expression")
	flag.Parse()

	if *hostsExp == "" {
		flag.Usage()
		os.Exit(1)
	}
	hosts, err := srg.ParseRange(*hostsExp)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var ports []string
	if *portsExp == "" {
		log.Printf("use default ports: %v", defaultPorts)
		ports = defaultPorts
	} else {
		ports, err = srg.ParseRange(*portsExp)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	for _, h := range hosts {
		for _, p := range ports {
			fmt.Printf("host = %s, port = %s\n", h, p)
			conn := fmt.Sprintf("%s:%s", h, p)

			client, err := redis.DialTimeout("tcp", conn, timeout)
			if err != nil {
				fmt.Println(err)
				continue
			}
			defer client.Close()

			info := client.Cmd("info")

			ret, err := parse(info)

			if err != nil {
				fmt.Println(err)
				continue
			}

			fmt.Println(ret["Clients"]["connected_clients"])
		}
	}
}

func parse(r *redis.Resp) (map[string]map[string]string, error) {

	ret := make(map[string]map[string]string)

	raw, err := r.Str()
	if err != nil {
		return nil, err
	}

	var section string
	for _, str := range strings.Split(raw, "\r\n") {
		if len(str) == 0 {
			continue
		}

		match := sectionPattern.FindStringSubmatch(str)
		if len(match) == 2 {
			section = match[1]
			continue
		}

		if ret[section] == nil {
			ret[section] = make(map[string]string)
		}
		parts := strings.Split(str, ":")
		ret[section][parts[0]] = parts[1]
	}

	return ret, nil
}
