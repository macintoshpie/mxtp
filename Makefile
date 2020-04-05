# functions to build
go_apps = bin/functions/hello bin/functions/fauna_db_example

bin/functions/%: functions/%/main.go
	cd $(<D) && go test .
	cd $(<D) && go build -o $(PWD)/bin/functions/$(*F) main.go

build: hugo $(go_apps)

hugo:
	hugo
