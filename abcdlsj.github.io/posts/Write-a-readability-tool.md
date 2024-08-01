---
title: "Build a website for readability at 10 minute"
date: 2023-10-22T11:51:37+08:00
tags:
  - Template
  - Readability
hide: false
hideToc: true
---

## Background
Many websites use various CSS styles, and some of them can be challenging to read. One way to improve readability is to use a browser extension. Alternatively, you can create your own website [WIKI - readability](https://www.wikiwand.com/en/Readability) tool. There are numerous language options and implementations available for this purpose. In this case, we will use `Go` to implement it. You will see how simple it is.

You can find all the code on my GitHub repository [here](https://github.com/abcdlsj/share/tree/master/go/readability)

## Do it
Start by reading the example code for [go-readability](https://github.com/go-shiori/go-readability) to understand its functionality and usage.

You can write the following code, which can be used as a command-line interface (CLI) tool.
```go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	readability "github.com/go-shiori/go-readability"
)

func extract(url string) readability.Article {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		log.Fatalf("failed to parse %s, %v\n", url, err)
	}

	fmt.Printf("Title   : %s\n", article.Title)
	fmt.Printf("Author  : %s\n", article.Byline)
	fmt.Printf("SiteName: %s\n", article.SiteName)
	fmt.Printf("Content : %s\n", article.Content)

	return article
}

func main() {
	var inputurl string
	flag.StringVar(&inputurl, "url", "", "URL")
	flag.Parse()

	if inputurl == "" {
		flag.Usage()
		os.Exit(1)
	}

	extract(inputurl)
}
```
It will work as expected.

However, there are instances where a terminal environment may not be available. In such cases, you can run it as a web server using Go's `http` package and `template` library to implement it. (In fact, I used `ChatGPT` to provide this demo. Cost 5 minute..)


**index.html**
```html
<!DOCTYPE html>
<html>
<head>
	<title>Extract Article Content</title>
</head>
<body>
	<h1>Extract Article Content</h1>
	<form action="/read" method="post">
		<label for="url">Enter URL:</label>
		<input type="text" id="url" name="url">
		<input type="submit" value="Extract">
	</form>
</body>
</html>
```

**article.html**
```html
<!DOCTYPE html>
<html>
<head>
	<title>Article Content</title>
</head>
<body>
	<h1>{{.Title}}</h1>
	{{if .ErrorMsg}}
		<p>{{.ErrorMsg}}</p>
	{{else}}
		<div class="content">
			{{.Content | safeHTML}}
		</div>
	{{end}}
</body>
</html>
```

**main.go**
```go
var (
    tmpl = ....
)
func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/read", read)

	log.Fatal(http.ListenAndServe(":8080", nil))
}


func index(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func read(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")

	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		render(w, Article{URL: url, ErrorMsg: err.Error()})
		return
	}

	render(w, Article{URL: url, Title: article.Title, Content: article.Content})
}

func render(w http.ResponseWriter, data Article) {
	err := tmpl.ExecuteTemplate(w, "article.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

Implementing it as a web server is also straightforward.

## Pack
Once we have the web server set up, it is generally a best practice to package it into a Docker image.

So you may need to pack the files `index.html` and `article.html` into a Docker image. In Go, you can use `embed` to embed these files into the binary. 
```go
var (
	//go:embed *.html
	tmplFiles embed.FS

	funcMap = template.FuncMap{
		"safeHTML": func(content string) template.HTML {
			return template.HTML(content)
		},
	}

	tmpl = template.Must(template.New("article.html").Funcs(funcMap).ParseFS(tmplFiles, "article.html", "index.html"))
)
```
`embed` will embed files at `compile` time.

## Docker

### Dockerfile
`Go` is a cross-platform programming language, you can copy the binary files into a base image to execute it. Here's an example of a Dockerfile:

Sample:
```docker
FROM alpine:latest
RUN mkdir /app
WORKDIR /app
COPY . .
CMD ["/app/readability"]
```
This Dockerfile copies the readability library into the image. Therefore, you need to compile it first using the command `CGO_ENABLED=0 GOOS=linux go build -o readability`.

I have also written a simple tool called nestg for packing binary files into a Docker image. You can find it at [share/nestg](https://github.com/abcdlsj/share/tree/master/go/nestg)

```
Usage of nestg:
  -b string
        go build flags
  -i string
        image name
  -p string
        port
```

Use `nestg`
```
nestg -i abcdlsj/readability -p 8080
```

### Run
After building the Docker image, you can run it using the following command:

`docker run -it --rm -p <HOST_PORT>:8080 abcdlsj/readability`

Now you can access the website at `http://localhost:<HOST_PORT>`.

That's all of things you can done within 10 minute. Then, I will optimize it!

## Optimize
### `Protect` double slashes

You can enhance the functionality by utilizing URL parameters, making it more useful. I attempted to add an HTTP URL to the path, but encountered a problem with the parser.

For example, when you make a request to `https://xxx.com/read/https://nautil.us/mirror-image-life-412729/`, the appended path becomes `https:/nautil.us/mirror-image-life-412729` due to the modification of `//` to `/`. This behavior is a result of the following explanation from the documentation:

> ServeMux also takes care of sanitizing the URL request path and the Host header, stripping the port number and redirecting any request containing . or .. elements or repeated slashes to an equivalent, cleaner URL.

This information was obtained from the documentation found at [net/http#ServeMux](https://pkg.go.dev/net/http#ServeMux)

I came across some useful links that discuss this issue:
- [Stack Overflow - URL-escaped parameter not resolving properly](https://stackoverflow.com/questions/55716545/url-escaped-parameter-not-resolving-properly)
- [Stack Overflow - How do I get Go's net/http package to stop removing double slashes?](https://stackoverflow.com/questions/51908277/how-do-i-get-gos-net-http-package-to-stop-removing-double-slashes)
- [Github issue - net/http: ServeMux forcibly cleans double forward slash in URLs even when behaving as a gateway](https://github.com/golang/go/issues/42244)

The suggested solution is to use `https://github.com/gorilla/mux` or implement your own `ServeMux`. By using gorilla/mux, you simply need to call `SkipClean(true)`. After doing this, the double slashes won't be removed.

### `http.Redirect` clean double slashes
However, I encountered another issue when using redirects after submitting a link form. The redirect operation also removes the double slashes from the URL path, and unfortunately, `gorilla/mux` does not support handling this situation.

Because `http.Redirect` call `url = path.Clean(url)` to clean the URL, it also will `clean` double slashes.

> Can see code [/src/net/http/server.go](https://github.com/golang/go/blob/8c92897e15d15fbc664cd5a05132ce800cf4017f/src/net/http/server.go#L2247C25-L2247C25)

Luckily, there is a workaround available. You can escape the `/` characters as `%2F` to prevent the removal of double slashes during the redirect process.

You can find this version of the code at [readability - 909fcb5e80fe](https://github.com/abcdlsj/share/tree/909fcb5e80fef9ecfaf68259ae98fb6694d3e984/go/readability). Please note that there are some additional adjustments made as well.

## Pictures
After adding some simple `CSS` to the `HTML template`, you will see the result:

<img alt="homepage" src="/static/img/readability_screenshot2.png" width="100%" style="border: 1px solid gray;">

<img alt="article page" src="/static/img/readability_screenshot.png" width="100%"  style="border: 1px solid gray;">

> 04/14/2024 Update: 
> Add recently viewed using `Redis` `list` to store, and `zset` store view count [commit](https://github.com/abcdlsj/share/commit/08837c71ff065791a400b976fa23ca2fd338d5bc).

<img alt="`recently` homepage" src="/static/img/readability_screenshot3.png" width="100%"  style="border: 1px solid gray;">

## Conclusion
This is a small weekend project. 

Thanks for reading!
