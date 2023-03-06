package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	SiteDescription string = "Enjoy Focus!"
	SiteTitle       string = "Enjoy Focus!"
	SiteURL         string = "http://127.0.0.1:8000"

	PostsDir  = flag.String("posts_dir", orEnv("POSTS_DIR", "example/posts"), "posts directory")
	OutputDir = flag.String("output_dir", orEnv("OUTPUT_DIR", "example/output"), "output directory")
	StaticDir = flag.String("static_dir", orEnv("STATIC_DIR", "example/static"), "static directory")

	md = goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			highlighting.Highlighting,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)

	funcMap = template.FuncMap{
		"joinTags": joinTags,
		"inTag":    inTag,
		"urlize":   urlize,
	}
)

type Site struct {
	SiteMeta

	PostsDir  string
	OutputDir string
	StaticDir string

	Posts []Post
	Tags  map[string]Tag
}

type SiteMeta struct {
	Title       string
	Description string
	URL         string
}

type Post struct {
	SiteMeta
	Meta map[string]interface{}
	Body string
	Name string
}

type Tag struct {
	SiteMeta
	Name   string
	Refers []string
}

func init() {
	flag.Parse()
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)
}

func orEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func (s *Site) gen() {
	s.parseAllPosts()

	for _, post := range s.Posts {
		log.Infof("post: %s", post.Name)
		for k, v := range post.Meta {
			log.Infof("meta: %s: %v", k, v)
		}
	}
	for _, tag := range s.Tags {
		log.Infof("tag: %s", tag.Name)
		for _, ref := range tag.Refers {
			log.Infof("ref: %s", ref)
		}
	}
	s.render()
}

func (s *Site) render() {
	s.renderIndex()

	s.renderPosts()
	s.renderTags()
}

func (s *Site) renderIndex() {
	tmpl := template.Must(template.New("index.html").Funcs(funcMap).ParseGlob("tmpl/index.html"))
	if err := render(tmpl, s, path.Join(s.OutputDir, "index.html")); err != nil {
		log.Fatal(err)
	}
}

func (s *Site) renderPosts() {
	tmpl := template.Must(template.New("single.html").Funcs(funcMap).ParseGlob("tmpl/single.html"))
	for _, post := range s.Posts {
		if err := render(tmpl, post, path.Join(s.OutputDir, "posts", urlize(post.Name)+".html")); err != nil {
			log.Fatal(err)
		}
	}
}

func (s *Site) sortPosts() {
	sort.Slice(s.Posts, func(i, j int) bool {
		return s.Posts[i].Meta["Date"].(string) > s.Posts[j].Meta["Date"].(string)
	})
}

func (s *Site) renderTags() {
	tmpl := template.Must(template.New("tag.html").Funcs(funcMap).ParseGlob("tmpl/tag.html"))
	for _, tag := range s.Tags {
		if err := render(tmpl, tag, path.Join(s.OutputDir, "tags", urlize(tag.Name)+".html")); err != nil {
			log.Fatal(err)
		}
	}
}

func render(tmpl *template.Template, data interface{}, name string) error {
	file, err := openWithCreatePath(name)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, data)
}

func openWithCreatePath(filename string) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(filename), 0755); err != nil {
		return nil, err
	}
	return os.Create(filename)
}

func (s *Site) parseAllPosts() {
	posts, err := os.ReadDir(s.PostsDir)
	if err != nil {
		log.Fatal("open posts dir error")
	}
	for _, post := range posts {
		if post.IsDir() {
			continue
		}
		data, err := os.ReadFile(path.Join(s.PostsDir, post.Name()))
		if err != nil {
			log.Fatal("open post file error")
		}
		urlizeName := urlize(post.Name()[:len(post.Name())-3])
		blog, err := parsePost(data, urlizeName)
		if err != nil {
			log.Fatal("parse post error")
		}
		s.Posts = append(s.Posts, blog)
		s.parseTags(blog.Meta["Tags"].([]interface{}), blog, urlizeName)
	}

	s.sortPosts()
}

func (s *Site) parseTags(tags []interface{}, post Post, urlizeName string) error {
	for _, tag := range tags {
		tagStr := fmt.Sprintf("%v", tag)
		if entry, ok := s.Tags[tagStr]; !ok {
			s.Tags[tagStr] = Tag{
				Name:   tagStr,
				Refers: []string{urlizeName},
				SiteMeta: SiteMeta{
					Title:       SiteTitle,
					Description: SiteDescription,
					URL:         SiteURL,
				},
			}
		} else {
			entry.Refers = append(entry.Refers, urlizeName)
			s.Tags[tagStr] = entry
		}
	}
	return nil
}

func parsePost(data []byte, urlizeName string) (Post, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert(data, &buf, parser.WithContext(context)); err != nil {
		panic(err)
	}
	metaData := meta.Get(context)
	return Post{
		SiteMeta: SiteMeta{
			Title:       SiteTitle,
			Description: SiteDescription,
			URL:         SiteURL,
		},
		Meta: metaData,
		Body: buf.String(),
		Name: urlizeName,
	}, nil
}

func main() {
	site := &Site{
		SiteMeta: SiteMeta{
			Title:       SiteTitle,
			Description: SiteDescription,
			URL:         SiteURL,
		},
		PostsDir:  *PostsDir,
		OutputDir: *OutputDir,
		StaticDir: *StaticDir,

		Posts: []Post{},
		Tags:  make(map[string]Tag),
	}

	site.gen()
}

func joinTags(tags []interface{}) string {
	var result bytes.Buffer
	for i, tag := range tags {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(fmt.Sprintf("<a href=\"%s/tags/%s.html\">%s</a>", SiteURL, url.QueryEscape(fmt.Sprint(tag)), tag))
	}
	return result.String()
}

func urlize(s string) string {
	return url.QueryEscape(s)
}

func inTag(tag string, tags []interface{}) bool {
	for _, t := range tags {
		if fmt.Sprint(t) == tag {
			return true
		}
	}
	return false
}
