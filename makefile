all: clean
	go run main.go -c config.toml
clean:
	rm -rf abcdlsj.github.io/public