.PHONY: run run-prod build clean

run:
	wails dev

run-prod: build
	./build/bin/forscadb

build:
	wails build

clean:
	rm -rf build/ frontend/dist/ frontend/node_modules/
