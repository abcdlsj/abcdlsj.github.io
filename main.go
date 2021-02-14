package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"text/template"

	"github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	PostsDir        = flag.String("posts_dir", orEnv("POSTS_DIR", "example/posts"), "posts directory")
	OutputDir       = flag.String("output_dir", orEnv("OUTPUT_DIR", "example/output"), "output directory")
	StaticDir       = flag.String("static_dir", orEnv("STATIC_DIR", "example/static"), "static directory")
	SiteURL         = flag.String("site_url", orEnv("SITE_URL", "http://127.0.0.1:8000"), "site url")
	SiteDescription = flag.String("site_description", orEnv("SITE_DESCRIPTION", "Enjoy Focus!"), "site description")
	SiteTitle       = flag.String("site_title", orEnv("SITE_TITLE", "Enjoy Focus!"), "site title")

	md = goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			highlighting.Highlighting,
			extension.GFM,
			extension.Footnote,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe()),
	)

	funcMap = template.FuncMap{
		"urlize": urlize,
	}

	//go:embed tmpl/*
	templateFiles embed.FS

	t *template.Template
)

var Posts []Post
var Tags map[string]Tag
var siteMeta SiteMeta

type SiteMeta struct {
	Title       string
	Description string
	URL         string
}

type Post struct {
	SiteMeta SiteMeta
	Meta     map[string]interface{}
	Body     string
	Name     string
}

type Tag struct {
	SiteMeta SiteMeta
	Name     string
	Refers   []string
}

func init() {
	flag.Parse()
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetLevel(log.DebugLevel)
}

func RenderIndex() {
	data := struct {
		SiteMeta SiteMeta
		Posts    []Post
	}{
		SiteMeta: siteMeta,
		Posts:    Posts,
	}
	if err := render(t, data, path.Join(*OutputDir, "index.html"), "index"); err != nil {
		log.Fatal(err)
	}
}

func RenderPosts() {
	for _, post := range Posts {
		if err := render(t, post, path.Join(*OutputDir, "posts", urlize(post.Name)+".html"), "single"); err != nil {
			log.Fatal(err)
		}
	}
}

func RenderTags() {
	for _, tag := range Tags {
		if err := render(t, tag, path.Join(*OutputDir, "tags", urlize(tag.Name)+".html"), "tag"); err != nil {
			log.Fatal(err)
		}
	}
}

func render(tmpl *template.Template, data interface{}, fPath, tName string) error {
	file, err := openWithCreatePath(fPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if tName != "" {
		return tmpl.ExecuteTemplate(file, tName, data)
	}
	return tmpl.Execute(file, data)
}

func openWithCreatePath(filename string) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(filename), 0755); err != nil {
		return nil, err
	}
	return os.Create(filename)
}

func ParseTags(tagNames []interface{}, post Post) error {
	for _, tag := range tagNames {
		tagStr := fmt.Sprintf("%v", tag)
		if entry, ok := Tags[tagStr]; !ok {
			Tags[tagStr] = Tag{
				Name:     tagStr,
				Refers:   []string{post.Name},
				SiteMeta: siteMeta,
			}
		} else {
			entry.Refers = append(entry.Refers, post.Name)
			Tags[tagStr] = entry
		}
	}
	return nil
}

func parsePost(data []byte, uName string) (Post, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert(data, &buf, parser.WithContext(context)); err != nil {
		log.Fatalf("failed to convert markdown, file: %s, err: %v", uName, err)
	}
	metaData := meta.Get(context)
	return Post{
		SiteMeta: siteMeta,
		Meta:     metaData,
		Body:     buf.String(),
		Name:     uName,
	}, nil
}

func main() {
	siteMeta = SiteMeta{
		Title:       *SiteTitle,
		Description: *SiteDescription,
		URL:         *SiteURL,
	}

	t = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFiles, "tmpl/*.html"))

	Tags = make(map[string]Tag)

	posts, err := os.ReadDir(*PostsDir)
	if err != nil {
		log.Fatal("open posts dir error ")
	}
	for _, p := range posts {
		if p.IsDir() {
			continue
		}
		data, err := os.ReadFile(path.Join(*PostsDir, p.Name()))
		if err != nil {
			log.Fatal("open post file error")
		}
		uName := urlize(p.Name()[:len(p.Name())-3])
		post, err := parsePost(data, uName)
		if err != nil {
			log.Fatal("parse post error")
		}
		Posts = append(Posts, post)
		ParseTags(post.Meta["Tags"].([]interface{}), post)
	}

	sort.Slice(Posts, func(i, j int) bool {
		return Posts[i].Meta["Date"].(string) > Posts[j].Meta["Date"].(string)
	})

	Renders(RenderIndex, RenderPosts, RenderTags)
	CpStaticDirToOutput()
}

func CpStaticDirToOutput() {
	outputStatic := path.Join(*OutputDir, "static")
	if err := os.RemoveAll(outputStatic); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(outputStatic, 0755); err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(*StaticDir, outputStatic); err != nil {
		log.Fatal(err)
	}
}

func Renders(fns ...func()) {
	for _, fn := range fns {
		fn()
	}
}

func urlize(s string) string {
	return url.QueryEscape(s)
}

func orEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
