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
	"github.com/parallellink/errr"
	"github.com/parallellink/srg"
)

var (
	defaultPorts   = []string{"11379", "11380", "11381"}
	timeout        = 1 * time.Second
	sectionPattern = regexp.MustCompile(`^# (.+)$`)
)

type RedisInfo struct {
	Info map[string]map[string]string
	Err  error
}

func main() {
	hostsExp := flag.String("h", "", "hosts expression")
	portsExp := flag.String("p", "", "ports expression")
	flag.Parse()

	if *hostsExp == "" {
		flag.Usage()
		os.Exit(1)
	}
	hosts, err := srg.ParseRange(*hostsExp)
	errr.ExitOnError(err)

	var ports []string
	if *portsExp == "" {
		log.Printf("use default ports: %v", defaultPorts)
		ports = defaultPorts
	} else {
		ports, err = srg.ParseRange(*portsExp)
		errr.ExitOnError(err)
	}

	for _, h := range hosts {
		for _, p := range ports {
			conn := fmt.Sprintf("%s:%s", h, p)

			ret := info(conn)

			if ret.Err != nil {
				fmt.Fprintf(os.Stderr, "%s : %v\n", conn, ret.Err)
			} else {
				fmt.Printf("%s : %v\n", conn, ret.Info["Clients"]["connected_clients"])
			}
		}
	}
}

func info(conn string) RedisInfo {
	client, err := redis.DialTimeout("tcp", conn, timeout)
	if err != nil {
		return RedisInfo{nil, err}
	}
	defer client.Close()

	info := client.Cmd("info")
	ret := parse(info)

	return ret
}

func parse(r *redis.Resp) RedisInfo {
	ret := RedisInfo{nil, nil}

	raw, err := r.Str()
	if err != nil {
		ret.Err = err
		return ret
	}

	ret.Info = make(map[string]map[string]string)
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

		if ret.Info[section] == nil {
			ret.Info[section] = make(map[string]string)
		}
		parts := strings.Split(str, ":")
		ret.Info[section][parts[0]] = parts[1]
	}

	return ret
}
