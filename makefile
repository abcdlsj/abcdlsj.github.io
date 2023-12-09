all: clean
	go run main.go -c config.toml
clean:
	rm -rf abcdlsj.github.io/public

serve: all
	python3 -m http.server 3000  --directory abcdlsj.github.io/public/