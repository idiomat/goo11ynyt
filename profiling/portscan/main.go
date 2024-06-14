package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
)

var (
	host                string
	ports               string
	numWorkers          int
	timeout             int
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
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of workers. Defaults to system's number of CPUs.")
	flag.IntVar(&timeout, "timeout", 5, "Timeout in seconds (default is 5).")
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
		fmt.Printf("failed to parse ports to scan: %s", err)
		os.Exit(1)
	}

	sem := semaphore.NewWeighted(int64(numWorkers))
	openPorts := make([]int, 0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	for _, port := range portsToScan {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("failed to acquire semaphore: %v", err)
			break
		}

		go func(port int) {
			defer sem.Release(1)
			sleepy(10)
			p := scan(host, port)
			if p != 0 {
				openPorts = append(openPorts, p)
			}
		}(port)
	}

	if err := sem.Acquire(ctx, int64(numWorkers)); err != nil {
		fmt.Printf("failed to acquire semaphore: %v", err)
	}

	fmt.Println()
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

func scan(host string, port int) int {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("%d CLOSED (%s)\n", port, err)
		return 0
	}
	conn.Close()
	return port
}

func sleepy(max int) {
	n := rand.IntN(max)
	time.Sleep(time.Duration(n) * time.Second)
}
