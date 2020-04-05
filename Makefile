# functions to build
go_apps = bin/functions/hello

bin/functions/%: functions/%.go
	go build -o $@ $<

build: hugo test $(go_apps)

hugo:
	hugo

test:
	cd functions && go test
