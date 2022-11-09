test:
	go build .
	./go-instrument -app my-service -filename internal/example/basic.go > internal/example/open_telemetry/basic.go
	cp internal/example/basic.go internal/example/open_telemetry_inplace/basic.go
	./go-instrument -app my-service -w -filename internal/example/open_telemetry_inplace/basic.go
