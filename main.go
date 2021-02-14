package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	PostsPathMatch    string = "posts"
	OutputDir         string = "public"
	PostTemplateFile  string = "static/post.html"
	IndexTemplateFile string = "static/index.html"
)

type OtherConfig struct {
	Author      string `yaml:"author"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Mail        string `yaml:"mail"`
	Github      string `yaml:"github"`
}

type Meta struct {
	Title   string
	Date    string
	Tags    []string
	Summary string
}

type Post struct {
	MetaData Meta
	Body     string
	FileName string
}

type Tag struct {
	Title string
	Link  string
	Count int
}

func createTagPostsMap(posts []Post) map[string][]Post {
	result := make(map[string][]Post)
	for _, post := range posts {
		for _, tag := range post.MetaData.Tags {
			key := strings.ToLower(tag)
			if result[key] == nil {
				result[key] = []Post{post}
			} else {
				result[key] = append(result[key], post)
			}
		}
	}

	return result
}

func getPostInfo(f string, name string) Post {
	fileRead, _ := ioutil.ReadFile(f)
	lines := strings.Split(string(fileRead), "\n")
	title := lines[1]
	date := lines[2]
	tags := strings.Split(lines[3], " ")
	summary := lines[4]
	body := strings.Join(lines[6:], "\n")
	htmlByte, err := markdown2HTML([]byte(body))
	if err != nil {
		log.Fatal("markdown2HTML error!")
	}

	return Post{Meta{title, date, tags, summary}, string(htmlByte), name}
}

func markdown2HTML(src []byte) ([]byte, error) {
	markdown := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithHardWraps()),
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"))))
	var buf bytes.Buffer
	if err := markdown.Convert(src, &buf); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

func getAllPosts() []Post {
	var ret []Post
	files, _ := filepath.Glob(PostsPathMatch + "/*")
	for _, f := range files {
		fname := strings.Replace(f, "posts/", "", -1)
		fname = strings.Replace(fname, ".md", "", -1)
		post := getPostInfo(f, fname)
		ret = append(ret, post)
	}
	return ret
}

func GenerateIndexHTML() {
	posts := getAllPosts()
	t, err := template.ParseFiles(IndexTemplateFile)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(path.Join(OutputDir, "index.html"))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	t.Execute(file, posts)
}

func GeneratePostsHTML() {
	files, _ := filepath.Glob(PostsPathMatch + "/*")
	t, _ := template.ParseFiles(PostTemplateFile)
	for _, f := range files {
		fname := strings.Replace(f, "posts/", "", -1)
		fname = strings.Replace(fname, ".md", "", -1)
		post := getPostInfo(f, fname)
		out, _ := os.Create(path.Join(OutputDir, PostsPathMatch, fname) + ".html")
		t.Execute(out, post)
	}
}

func GenerateTagsHTML(posts []Post) {
	tagsMap := createTagPostsMap(posts)
	_ = tagsMap
}

func main() {
	GenerateIndexHTML()
	GeneratePostsHTML()
}
