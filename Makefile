# functions to build
go_apps = bin/functions/jockey
go_lib = functions/bouncer/bouncer.go functions/mxtpdb/mxtpdb.go

./bin/functions/%: functions/%/main.go $(go_lib)
	cd $(<D) && go test .
	cd $(<D) && GOOS=linux go build -o $(PWD)/bin/functions/$(*F) .

clean:
	rm -rf bin/functions

build: $(go_apps)
