all: clean
	go run main.go
clean:
	rm -rf abcdlsj.github.io/public/posts
	rm -rf abcdlsj.github.io/public/tags
	rm -rf abcdlsj.github.io/public/categories
	rm abcdlsj.github.io/public/index.html