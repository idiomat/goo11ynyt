package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
)

var (
	host                string
	ports               string
	numWorkers          int
	profileDir          string
	cpuprofile          string
	memprofile          string
	blockprofile        string
	mutexprofile        string
	goroutineprofile    string
	threadcreateprofile string
)

func init() {
	flag.StringVar(&host, "host", "127.0.0.1", "Host to scan.")
	flag.StringVar(&ports, "ports", "5000-5500", "Port(s) (e.g. 80, 22-100).")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of workers (defaults to # of logical CPUs).")
	flag.StringVar(&profileDir, "profile-dir", "profiling/profiles", "Directory to store profiles.")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&memprofile, "memprofile", "", "write memory profile to file")
	flag.StringVar(&blockprofile, "blockprofile", "", "write block profile to file")
	flag.StringVar(&mutexprofile, "mutexprofile", "", "write mutex profile to file")
	flag.StringVar(&goroutineprofile, "goroutineprofile", "", "write goroutine profile to file")
	flag.StringVar(&threadcreateprofile, "threadcreateprofile", "", "write threadcreate profile to file")
}

func main() {
	flag.Parse()

	if cpuprofile != "" {
		pf, err := os.Create(fmt.Sprintf("%s/%s.cpu.pprof", profileDir, cpuprofile))
		if err != nil {
			log.Fatalln(err)
		}
		defer pf.Close()

		if err := pprof.StartCPUProfile(pf); err != nil {
			log.Fatalln(err)
		}
		defer pprof.StopCPUProfile()
	}

	if blockprofile != "" {
		pf, err := os.Create(fmt.Sprintf("%s/%s.block.pprof", profileDir, blockprofile))
		if err != nil {
			log.Fatalln(err)
		}
		defer pf.Close()
		runtime.SetBlockProfileRate(1)             // enables block profiling
		defer pprof.Lookup("block").WriteTo(pf, 0) //nolint:errcheck
	}

	if mutexprofile != "" {
		pf, err := os.Create(fmt.Sprintf("%s/%s.mutex.pprof", profileDir, mutexprofile))
		if err != nil {
			log.Fatalln(err)
		}
		defer pf.Close()
		runtime.SetMutexProfileFraction(1)         // enables mutex profiling
		defer pprof.Lookup("mutex").WriteTo(pf, 0) //nolint:errcheck
	}

	if goroutineprofile != "" {
		pf, err := os.Create(fmt.Sprintf("%s/%s.goroutine.pprof", profileDir, goroutineprofile))
		if err != nil {
			log.Fatalln(err)
		}
		defer pf.Close()
		defer pprof.Lookup("goroutine").WriteTo(pf, 0) //nolint:errcheck
	}

	portsToScan, err := parsePortsToScan(ports)
	if err != nil {
		fmt.Printf("failed to parse ports to scan: %s\n", err)
		os.Exit(1)
	}

	tcpScanner, err := NewTCPScanner(host, numWorkers, &net.Dialer{})
	if err != nil {
		fmt.Printf("failed to create TCP scanner: %s\n", err)
		os.Exit(1)
	}

	openPorts, err := tcpScanner.Scan(portsToScan)
	if err != nil {
		fmt.Printf("failed to scan ports: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("RESULTS")
	sort.Ints(openPorts)
	for _, p := range openPorts {
		fmt.Printf("%d - open\n", p)
	}

	if memprofile != "" {
		pf, err := os.Create(fmt.Sprintf("%s/%s.mem.pprof", profileDir, memprofile))
		if err != nil {
			log.Fatalln(err)
		}
		defer pf.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(pf); err != nil {
			log.Fatalln(err)
		}
	}
}

func parsePortsToScan(portsFlag string) ([]int, error) {
	p, err := strconv.Atoi(portsFlag)
	if err == nil {
		return []int{p}, nil
	}

	ports := strings.Split(portsFlag, "-")
	if len(ports) != 2 {
		return nil, errors.New("unable to determine port(s) to scan")
	}

	minPort, err := strconv.Atoi(ports[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to a valid port number", ports[0])
	}

	maxPort, err := strconv.Atoi(ports[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert %s to a valid port number", ports[1])
	}

	if minPort <= 0 || maxPort <= 0 {
		return nil, fmt.Errorf("port numbers must be greater than 0")
	}

	var results []int
	for p := minPort; p <= maxPort; p++ {
		results = append(results, p)
	}
	return results, nil
}
