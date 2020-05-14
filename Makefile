# functions to build
go_apps = bin/functions/hello bin/functions/leagues

./bin/functions/%: functions/%/main.go
	cd $(<D) && go test .
	cd $(<D) && go build -o $(PWD)/bin/functions/$(*F) .

clean:
	rm -rf bin/functions

build: hugo $(go_apps)

hugo:
	hugo
