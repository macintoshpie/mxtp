go_apps = bin/hello

bin/%: functions/%.go
	go build -o $@ $<

build: test $(go_apps)

test:
	cd functions && go test
