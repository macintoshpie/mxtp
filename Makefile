# functions to build
go_apps = bin/hello

bin/%: functions/%.go
	go build -o $@ $<

build: hugo test $(go_apps)

hugo:
	hugo

test:
	cd functions && go test
