package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/BurntSushi/toml"
	d2 "github.com/FurqanSoftware/goldmark-d2"
	"github.com/abcdlsj/cr"
	"github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/toc"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
)

type CfgVar struct {
	URL         string `toml:"url"`
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Menus       []struct {
		Slug string `toml:"slug"`
		Name string `toml:"name"`
		URL  string `toml:"url"`
	} `toml:"menus"`
	Build struct {
		Posts  string `toml:"posts"`
		Output string `toml:"output"`
		Static string `toml:"static"`
	} `toml:"build"`
}

func mustCfg(f string) CfgVar {
	var cfg CfgVar
	if _, err := toml.DecodeFile(f, &cfg); err != nil {
		log.Fatalf("decode config file error: %v", err)
	}
	return cfg
}

var (
	cfgFile = flag.String("c", "config.toml", "config file")
	cfgVar  CfgVar

	md = goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			highlighting.Highlighting,
			extension.GFM,
			extension.Footnote,
			&d2.Extender{
				Layout:  d2dagrelayout.DefaultLayout,
				ThemeID: d2themescatalog.NeutralDefault.ID,
				Sketch:  true,
			},
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe()),
	)

	tocMd = goldmark.New(
		goldmark.WithExtensions(
			&toc.Extender{
				MaxDepth: 4,
				Title:    "TOC",
			},
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)

	funcMap = template.FuncMap{
		"urlize": urlize,
	}

	//go:embed tmpl/*
	templateFiles embed.FS

	t *template.Template

	Posts     []Post
	WipPosts  []Post
	TagMap    map[string]Tag
	AboutPost Post
)

type PostMeta struct {
	Title string   `yaml:"title"`
	Date  string   `yaml:"date"`
	Tags  []string `yaml:"tags"`
	Hide  bool     `yaml:"hide"`
	Menus []string `yaml:"menus"`
	Wip   bool     `yaml:"wip"`
}

func unmarshalPostMeta(meta map[string]interface{}) PostMeta {
	return PostMeta{
		Title: meta["title"].(string),
		Date:  orStr(meta["date"].(string), "1970-01-01"),
		Tags:  getMetaStrs(meta, "tags"),
		Hide:  meta["hide"].(bool),
		Menus: getMetaStrs(meta, "menus"),
		Wip:   getMetaBool(meta, "wip"),
	}
}

type Post struct {
	Site       CfgVar
	Meta       PostMeta
	Body       string
	Uname      string
	TocContent string
}

type Tag struct {
	Site   CfgVar
	Name   string
	Refers []Refer
}

type Refer struct {
	Title string
	Uname string
	Meta  PostMeta
}

func init() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetLevel(log.DebugLevel)
}

func RenderIndex() {
	posts := make([]Post, 0, len(Posts))

	for _, post := range Posts {
		if !post.Meta.Hide {
			posts = append(posts, post)
		}
	}

	data := struct {
		Site     CfgVar
		Posts    []Post
		WipPosts []Post
	}{
		Site:     cfgVar,
		Posts:    posts,
		WipPosts: WipPosts,
	}

	if err := render(t, data, path.Join(cfgVar.Build.Output, "index.html"), "index"); err != nil {
		log.Fatal(err)
	}
}

func RenderPosts() {
	for _, post := range append(Posts, WipPosts...) {
		if err := render(t, post, path.Join(cfgVar.Build.Output, "posts", post.Uname+".html"), "single"); err != nil {
			log.Fatal(err)
		}
	}
}

func RenderTags() {
	for _, tag := range TagMap {
		if err := render(t, tag, path.Join(cfgVar.Build.Output, "tags", urlize(tag.Name)+".html"), "tag"); err != nil {
			log.Fatal(err)
		}
	}
}

func RenderAbout() {
	if err := render(t, AboutPost, path.Join(cfgVar.Build.Output, "about/index.html"), "about"); err != nil {
		log.Fatal(err)
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

func parseTags(tagNames []string, post Post) error {
	tagRefer := Refer{
		Title: post.Meta.Title,
		Uname: post.Uname,
		Meta:  post.Meta,
	}

	for _, tag := range tagNames {
		if entry, ok := TagMap[tag]; !ok {
			TagMap[tag] = Tag{
				Name:   tag,
				Refers: []Refer{tagRefer},
				Site:   cfgVar,
			}
		} else {
			entry.Refers = append(entry.Refers, tagRefer)
			TagMap[tag] = entry
		}
	}

	return nil
}

func parsePost(data []byte, cleanName string) (Post, error) {
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert(data, &buf, parser.WithContext(context)); err != nil {
		log.Fatalf("failed to convert markdown, file: %s, err: %v", cleanName, err)
	}

	doc := tocMd.Parser().Parse(text.NewReader(data))

	tree, err := toc.Inspect(doc, data)
	if err != nil {
		log.Fatalf("failed to inspect toc, file: %s, err: %v", cleanName, err)
	}

	var tocBuf bytes.Buffer

	if len(tree.Items) != 0 {
		treeList := toc.RenderList(tree)

		if err := tocMd.Renderer().Render(&tocBuf, data, treeList); err != nil {
			log.Fatalf("failed to render toc, file: %s, err: %v", cleanName, err)
		}
	}

	return Post{
		Site:       cfgVar,
		Meta:       unmarshalPostMeta(meta.Get(context)),
		Body:       buf.String(),
		Uname:      urlize(cleanName),
		TocContent: tocBuf.String(),
	}, nil
}

func main() {
	cfgVar = mustCfg(*cfgFile)

	t = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFiles, "tmpl/*.html"))

	TagMap = make(map[string]Tag)

	posts, err := getAllFiles(cfgVar.Build.Posts)
	if err != nil {
		log.Fatal("open posts dir error ")
	}

	for _, p := range posts {
		fmt.Printf("Load post: %s\n", cr.PLYellow(p))
		fdata, err := os.ReadFile(p)
		if err != nil {
			log.Fatal("open post file error")
		}

		base := filepath.Base(p)
		cleanName := base[:len(base)-len(filepath.Ext(base))]
		post, err := parsePost(fdata, strings.ToLower(cleanName))

		if err != nil {
			log.Fatal("parse post error")
		}

		if post.Meta.Menus != nil && post.Meta.Menus[0] == "about" {
			AboutPost = post
			continue
		}

		if post.Meta.Wip {
			fmt.Printf("Parsed wip post: %s\n", cr.PLYellow(p))
			WipPosts = append(WipPosts, post)
		} else {
			fmt.Printf("Parsed post: %s\n", cr.PLGreen(p))
			Posts = append(Posts, post)
		}

		if !post.Meta.Hide {
			parseTags(post.Meta.Tags, post)
		}
	}

	sort.Slice(Posts, func(i, j int) bool {
		return dateCompare(Posts[i].Meta.Date, Posts[j].Meta.Date)
	})

	sortTagMap := make(map[string]Tag)
	for k, v := range TagMap {
		sort.Slice(v.Refers, func(i, j int) bool {
			return dateCompare(v.Refers[i].Meta.Date, v.Refers[j].Meta.Date)
		})

		sortTagMap[k] = v
	}

	TagMap = sortTagMap

	Renders(RenderIndex, RenderPosts, RenderTags, RenderAbout)
	CpStaticDirToOutput()

	fmt.Println(cr.PLCyan("All done!!!"))
}

func dateCompare(a, b string) bool {
	layout := "2006-01-02T15:04:05Z07:00"

	ta, err := time.Parse(layout, a)
	if err != nil {
		log.Fatal(err)
		return false
	}
	tb, err := time.Parse(layout, b)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return ta.After(tb)
}

func getMetaStrs(meta map[string]interface{}, key string) []string {
	if val, ok := meta[key]; ok {
		strs := val.([]interface{})
		var strsStrs []string
		for _, str := range strs {
			strsStrs = append(strsStrs, fmt.Sprintf("%v", str))
		}
		return strsStrs
	}
	return nil
}

func getMetaBool(meta map[string]interface{}, key string) bool {
	if val, ok := meta[key]; ok {
		return val.(bool)
	}
	return false
}

func CpStaticDirToOutput() {
	outputStatic := path.Join(cfgVar.Build.Output, "static")
	if err := os.RemoveAll(outputStatic); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(outputStatic, 0755); err != nil {
		log.Fatal(err)
	}
	if err := copy.Copy(cfgVar.Build.Static, outputStatic); err != nil {
		log.Fatal(err)
	}

	fmt.Println(cr.PLCyan("Copy static dir success"))
}

func Renders(fns ...func()) {
	for _, fn := range fns {
		fn()
	}

	fmt.Println(cr.PLCyan("Render success"))
}

func urlize(s string) string {
	return strings.ToLower(url.QueryEscape(s))
}

func orStr(s string, dv string) string {
	if s != "" {
		return s
	}
	return dv
}

func getAllFiles(dir string) ([]string, error) {
	var result []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			result = append(result, path)
		}
		return nil
	})

	return result, nil
}
