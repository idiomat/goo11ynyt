wordlens-benchmark-unoptimized:
	-@mkdir ./benchmarking/wordlens/benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./benchmarking/wordlens/benchmarks/unoptimized.mem.prof \
		-cpuprofile ./benchmarking/wordlens/benchmarks/unoptimized.cpu.prof \
		./benchmarking/wordlens/unoptimized | tee ./benchmarking/wordlens/benchmarks/unoptimized.bench.txt

wordlens-pprof-unoptimized-cpu:
	go tool pprof -http=:8080 ./benchmarking/wordlens/benchmarks/unoptimized.cpu.prof

wordlens-pprof-unoptimized-mem:
	go tool pprof -http=:8080 ./benchmarking/wordlens/benchmarks/unoptimized.mem.prof

wordlens-benchmark-optimized:
	-@mkdir ./benchmarking/wordlens/benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./benchmarking/wordlens/benchmarks/optimized.mem.prof \
		-cpuprofile ./benchmarking/wordlens/benchmarks/optimized.cpu.prof \
		./benchmarking/wordlens/optimized | tee ./benchmarking/wordlens/benchmarks/optimized.bench.txt

install-benchstat:
	go install golang.org/x/perf/cmd/benchstat@latest

wordlens-benchstat-unoptimized-vs-optimized:
	benchstat -filter ".unit:(sec/op)" \
		unoptimized=./benchmarking/wordlens/benchmarks/unoptimized.bench.txt \
		optimized=./benchmarking/wordlens/benchmarks/optimized.bench.txt

dir-for-e1-benchmarks:
	-@mkdir ./e1/benchmarks

e1-benchmark-sequential: dir-for-e1-benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./e1/benchmarks/sequential.mem.prof \
		-cpuprofile ./e1/benchmarks/sequential.cpu.prof \
		./e1 \
		-concurrent=false | tee ./e1/benchmarks/sequential.bench.txt

e1-benchmark-concurrent: dir-for-e1-benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./e1/benchmarks/concurrent.mem.prof \
		-cpuprofile ./e1/benchmarks/concurrent.cpu.prof \
		./e1 \
		-concurrent=true | tee ./e1/benchmarks/concurrent.bench.txt

e1-benchstat-sequential-vs-concurrent:
	benchstat \
		sequential=./e1/benchmarks/sequential.bench.txt \
		concurrent=./e1/benchmarks/concurrent.bench.txt

e1-pprof-sequential-cpu:
	go tool pprof -http=:8080 ./e1/benchmarks/sequential.cpu.prof

e1-pprof-sequential-mem:
	go tool pprof -http=:8081 ./e1/benchmarks/sequential.mem.prof

e1-pprof-concurrent-cpu:
	go tool pprof -http=:8082 ./e1/benchmarks/concurrent.cpu.prof

e1-pprof-concurrent-mem:
	go tool pprof -http=:8083 ./e1/benchmarks/concurrent.mem.prof

dir-for-e2-benchmarks:
	-@mkdir ./e2/benchmarks

e2-benchmark-sequential: dir-for-e2-benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./e2/benchmarks/sequential.mem.prof \
		-cpuprofile ./e2/benchmarks/sequential.cpu.prof \
		./e2 \
		-technique=sequential | tee ./e2/benchmarks/sequential.bench.txt

e2-benchmark-mutex: dir-for-e2-benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./e2/benchmarks/mutex.mem.prof \
		-cpuprofile ./e2/benchmarks/mutex.cpu.prof \
		./e2 \
		-technique=mutex | tee ./e2/benchmarks/mutex.bench.txt

e2-benchmark-channel: dir-for-e2-benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./e2/benchmarks/channel.mem.prof \
		-cpuprofile ./e2/benchmarks/channel.cpu.prof \
		./e2 \
		-technique=channel | tee ./e2/benchmarks/channel.bench.txt

e2-benchmark-workers: dir-for-e2-benchmarks
	go test -bench=BenchmarkFindPalindromes \
		-count=10 \
		-benchmem \
		-memprofile ./e2/benchmarks/workers.mem.prof \
		-cpuprofile ./e2/benchmarks/workers.cpu.prof \
		./e2 \
		-technique=workers | tee ./e2/benchmarks/workers.bench.txt

e2-benchstat-seq-vs-mutex-vs-channel-vs-workers:
	benchstat \
		seq=./e2/benchmarks/sequential.bench.txt \
		mutex=./e2/benchmarks/mutex.bench.txt \
		channel=./e2/benchmarks/channel.bench.txt \
		workers=./e2/benchmarks/workers.bench.txt

PROFILE_DIR ?= ./profiling/profiles
profiles-dir:
	-@mkdir $(PROFILE_DIR)

profile-portscan: profiles-dir
	go run ./profiling/portscan/main.go \
		-ports=5430-5440 \
		-timeout=15 \
		-profile-dir=$(PROFILE_DIR) \
		-cpuprofile=semaphore \
		-memprofile=semaphore \
		-blockprofile=semaphore \
		-mutexprofile=semaphore \
		-goroutineprofile=semaphore

profile-e3: profiles-dir
	go run ./e3/./... \
		-ports=5430-5440 \
		-profile-dir=$(PROFILE_DIR) \
		-cpuprofile=fanout-fanin \
		-memprofile=fanout-fanin \
		-blockprofile=fanout-fanin \
		-mutexprofile=fanout-fanin \
		-goroutineprofile=fanout-fanin

TRACE_DIR ?= ./tracing/traces
traces-dir:
	-@mkdir $(TRACE_DIR)

trace-portscan: traces-dir
	go run ./tracing/portscan/main.go \
		-ports=5430-5440 \
		-timeout=15 \
		-trace-dir=$(TRACE_DIR)

run-metrics:
	go run metrics/*.go

run-otel:
	go run otel/*.go -book=data/pg2680.txt