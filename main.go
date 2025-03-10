package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	d2 "github.com/FurqanSoftware/goldmark-d2"
	"github.com/abcdlsj/cr"
	"github.com/huichen/sego"
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
	URL         string   `toml:"url"`
	Title       string   `toml:"title"`
	Description string   `toml:"description"`
	Homepage    string   `toml:"homepage"`
	Keywords    []string `toml:"keywords"`
	Menus       []struct {
		Slug string `toml:"slug"`
		Name string `toml:"name"`
		URL  string `toml:"url"`
		Hide bool   `toml:"hide"`
		Dir  bool   `toml:"dir"`
	} `toml:"menus"`
	Build struct {
		Posts  string `toml:"posts"`
		Output string `toml:"output"`
		Static string `toml:"static"`
	} `toml:"build"`
	Hosts []struct {
		Name   string `toml:"name"`
		Source string `toml:"source"`
		Output string `toml:"output"`
		Type   string `toml:"type"`
		Header string `toml:"header"`
	} `toml:"host"`
	Author string `toml:"author"`
}

func mustCfg(f string) CfgVar {
	var cfg CfgVar
	if _, err := toml.DecodeFile(f, &cfg); err != nil {
		log.Fatalf("decode config file error: %v", err)
	}

	cfg.Homepage = mustmd(cfg.Homepage)
	return cfg
}

func mustmd(f string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(f), &buf); err != nil {
		log.Fatalf("convert markdown error: %v", err)
	}
	return buf.String()
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
			extension.CJK,
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
				MaxDepth: 3,
				Title:    "",
			},
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)

	funcMap = template.FuncMap{
		"urlize": urlize,
		"add":    func(a, b int) int { return a + b },
		"day": func(s string) string {
			t, err := time.Parse("2006-01-02T15:04:05Z07:00", s)
			if err != nil {
				log.Fatal(err)
			}
			return t.Format("2006-01-02")
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"truncate": func(s string, n int) string {
			if len(s) > n {
				return s[:n] + "..."
			}
			return s
		},
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
	}

	//go:embed tmpl/*
	templateFiles embed.FS

	t *template.Template

	sm sego.Segmenter

	Posts     []Post
	WipPosts  []Post
	TagMap    map[string]Tag
	AboutPost Post
)

type PostMeta struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Tags        []string `yaml:"tags"`
	Hide        bool     `yaml:"hide"`
	Menus       []string `yaml:"menus"`
	Wip         bool     `yaml:"wip"`
	TocPosition string   `yaml:"tocPosition"`
	HideToc     bool     `yaml:"hideToc"`
	Hero        string   `yaml:"hero"`
	Description string   `yaml:"description"`
	Languages   []string `yaml:"languages"`
}

func unmarshalPostMeta(meta map[string]interface{}) PostMeta {
	return PostMeta{
		Title:       meta["title"].(string),
		Date:        orStr(meta["date"].(string), "1970-01-01"),
		Tags:        getMetaStrs(meta, "tags"),
		Hide:        meta["hide"].(bool),
		Menus:       getMetaStrs(meta, "menus"),
		Wip:         getMetaBool(meta, "wip"),
		TocPosition: orStr(getMetaStr(meta, "tocPosition"), ""),
		HideToc:     getMetaBool(meta, "hideToc"),
		Hero:        orStr(getMetaStr(meta, "hero"), ""),
		Description: orStr(getMetaStr(meta, "description"), ""),
		Languages:   orStrs(getMetaStrs(meta, "languages"), []string{"en"}),
	}
}

type Post struct {
	Site       CfgVar
	Meta       PostMeta
	Body       string
	Uname      string
	TocContent string
	MDData     string
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

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type FollowChanllenge struct {
	XMLName xml.Name `xml:"follow_challenge"`
	FeedId  string   `xml:"feedId"`
	UserId  string   `xml:"userId"`
}

type Channel struct {
	Title           string           `xml:"title"`
	Link            string           `xml:"link"`
	Description     string           `xml:"description"`
	Items           []Item           `xml:"item"`
	FollowChallenge FollowChanllenge `xml:"follow_challenge"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type SearchIndex struct {
	Words map[string][]string `json:"words"`
}

func init() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetLevel(log.DebugLevel)

	smLoadDict()
}

func RenderIndex() {
	posts := make([]Post, 0, len(Posts))
	for _, post := range Posts {
		if !post.Meta.Hide {
			posts = append(posts, post)
		}
	}

	wipPosts := make([]Post, 0, len(WipPosts))
	for _, post := range WipPosts {
		if !post.Meta.Hide {
			wipPosts = append(wipPosts, post)
		}
	}

	data := struct {
		Site     CfgVar
		Posts    []Post
		WipPosts []Post
	}{
		Site:     cfgVar,
		Posts:    posts,
		WipPosts: wipPosts,
	}

	if err := render(t, data, path.Join(cfgVar.Build.Output, "index.html"), "index"); err != nil {
		log.Fatal(err)
	}

	if err := render(t, data, path.Join(cfgVar.Build.Output, "posts/index.html"), "posts"); err != nil {
		log.Fatal(err)
	}
}

func RenderHostsIndex() {
	data := struct {
		Site CfgVar
	}{
		Site: cfgVar,
	}
	if err := render(t, data, path.Join(cfgVar.Build.Output, "hosts/index.html"), "hosts"); err != nil {
		log.Fatal(err)
	}
}

func RenderPosts() {
	for _, post := range append(Posts, WipPosts...) {
		post.Site.Title = post.Meta.Title
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

	meta := unmarshalPostMeta(meta.Get(context))

	post := Post{
		Site:   cfgVar,
		Meta:   meta,
		Body:   buf.String(),
		Uname:  generateUniqueURL(cleanName),
		MDData: string(data),
	}

	if !post.Meta.HideToc {
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

		post.TocContent = strings.TrimSuffix(strings.TrimPrefix(tocBuf.String(), "<ul>"), "</ul>")
	}

	return post, nil
}

func generateUniqueURL(name string) string {
	return urlize(name)
}

func GenerateRSS() error {
	channel := Channel{
		Title:       cfgVar.Title,
		Link:        cfgVar.URL,
		Description: cfgVar.Description,
		FollowChallenge: FollowChanllenge{
			FeedId: "81944482269007872",
			UserId: "54069612848210944",
		},
	}

	for _, post := range Posts {
		if post.Meta.Hide {
			continue
		}
		pubDate, _ := time.Parse("2006-01-02T15:04:05Z07:00", post.Meta.Date)
		item := Item{
			Title:       post.Meta.Title,
			Link:        cfgVar.URL + "/posts/" + post.Uname + ".html",
			Description: post.Meta.Title,
			PubDate:     pubDate.Format(time.RFC1123Z),
		}
		channel.Items = append(channel.Items, item)
	}

	rss := RSS{
		Version: "2.0",
		Channel: channel,
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return err
	}

	rssFile, err := os.Create(path.Join(cfgVar.Build.Output, "rss.xml"))
	if err != nil {
		return err
	}
	defer rssFile.Close()

	rssFile.WriteString(xml.Header)
	rssFile.Write(output)

	return nil
}

func optimizeImages(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".png")) {
			// img, err := imaging.Open(path)
			// if err != nil {
			// 	return err
			// }
			// resized := imaging.Resize(img, 800, 0, imaging.Lanczos)
			// return imaging.Save(resized, path)

			// 啥也不做
		}
		return nil
	})
}

func generateSitemap() error {
	sitemap := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
	}

	for _, post := range Posts {
		sitemap = append(sitemap, fmt.Sprintf(`  <url>
    <loc>%s/posts/%s.html</loc>
    <lastmod>%s</lastmod>
  </url>`, cfgVar.URL, post.Uname, post.Meta.Date))
	}

	sitemap = append(sitemap, `</urlset>`)

	return os.WriteFile(path.Join(cfgVar.Build.Output, "sitemap.xml"), []byte(strings.Join(sitemap, "\n")), 0644)
}

func main() {
	cfgVar = mustCfg(*cfgFile)

	t = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFiles, "tmpl/*.html"))

	TagMap = make(map[string]Tag)

	CpHostsDirToOutput()

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
		post, err := parsePost(fdata, cleanName)

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

	CreateMenuOutputDirs()
	Renders(RenderIndex, RenderPosts, RenderTags, RenderAbout, RenderHostsIndex)
	CpStaticDirToOutput()

	if err := GenerateRSS(); err != nil {
		log.Fatal(err)
	}

	if err := optimizeImages(path.Join(cfgVar.Build.Output, "static")); err != nil {
		log.Fatal(err)
	}

	if err := generateSitemap(); err != nil {
		log.Fatal(err)
	}

	if err := generateSearchIndex(); err != nil {
		log.Fatal(err)
	}

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

func getMetaStr(meta map[string]interface{}, key string) string {
	if val, ok := meta[key]; ok {
		return fmt.Sprintf("%v", val)
	}

	return ""
}

func getMetaBool(meta map[string]interface{}, key string) bool {
	if val, ok := meta[key]; ok {
		return val.(bool)
	}
	return false
}

func CreateMenuOutputDirs() {
	fmt.Println(cr.PLCyan("Create menu output dirs"))
	for _, menu := range cfgVar.Menus {
		if !menu.Dir {
			continue
		}
		fmt.Printf("Create menu output dir: %s\n", menu.URL)
		outputMenu := path.Join(cfgVar.Build.Output, menu.URL)
		if err := os.MkdirAll(outputMenu, 0755); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println(cr.PLCyan("Create menu output dirs success"))
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

func CpHostsDirToOutput() {
	fmt.Println(cr.PLCyan("Copy hosts dir to output"))
	for _, host := range cfgVar.Hosts {
		if host.Type == "static" {
			fmt.Printf("Copy host: %s\n", host.Name)
			outputHost := path.Join(cfgVar.Build.Output, host.Output)
			if err := os.RemoveAll(outputHost); err != nil {
				log.Fatal(err)
			}
			if err := os.MkdirAll(outputHost, 0755); err != nil {
				log.Fatal(err)
			}
			if err := copy.Copy(host.Source, outputHost); err != nil {
				log.Fatal(err)
			}
		} else if host.Type == "render_post" {
			host.Source = os.ExpandEnv(host.Source)
			fmt.Printf("Copy file to posts: %s\n", host.Name)
			filename := path.Base(host.Source)
			output := path.Join(cfgVar.Build.Posts, filename)

			content, err := os.ReadFile(host.Source)
			if err != nil {
				log.Fatal(err)
			}

			content = []byte(host.Header + "\n" + string(content))
			os.WriteFile(output, content, 0644)
		}
	}

	fmt.Println(cr.PLCyan("Copy hosts dir success"))
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

func orStrs(s []string, dv []string) []string {
	if len(s) != 0 {
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

func generateSearchIndex() error {
	index := SearchIndex{Words: make(map[string][]string)}

	for _, post := range Posts {
		fullText := post.Meta.Title + " " + stripHTML(post.MDData)
		words := analyze(fullText)

		for _, word := range words {
			word = strings.ToLower(word)
			if !contains(index.Words[word], post.Uname) {
				index.Words[word] = append(index.Words[word], post.Uname)
			}
		}
	}

	jsonData, err := json.Marshal(index)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(cfgVar.Build.Output, "search-index.json"), jsonData, 0644)
}

func stripHTML(input string) string {
	return strings.ReplaceAll(strings.ReplaceAll(input, "<", " "), ">", " ")
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func smLoadDict() {
	sm.LoadDictionary("dictionary.txt")
}

func analyze(text string) []string {
	segments := sm.Segment([]byte(text))
	words := sego.SegmentsToSlice(segments, false)

	var filteredWords []string
	for _, word := range words {
		if isNumeric(word) {
			continue
		}

		if isImageFile(word) {
			continue
		}

		if utf8.RuneCountInString(word) < 2 ||
			utf8.RuneCountInString(word) > 10 ||
			len(word) < 2 || len(word) > 10 {
			continue
		}

		if isStopWord(word) {
			continue
		}

		if isGibberish(word) {
			continue
		}

		filteredWords = append(filteredWords, word)
	}

	return filteredWords
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isImageFile(s string) bool {
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	for _, ext := range extensions {
		if strings.HasSuffix(strings.ToLower(s), ext) {
			return true
		}
	}
	return false
}

func isStopWord(s string) bool {
	stopWords := map[string]bool{
		"的": true, "了": true, "和": true, "是": true, "就": true,
		"在": true, "也": true, "为": true, "而": true, "以": true,
		"与": true, "或": true, "一": true, "把": true, "但": true,
	}
	return stopWords[s]
}

func isGibberish(s string) bool {
	nonAlphaNumCount := 0
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			nonAlphaNumCount++
		}
	}
	if float64(nonAlphaNumCount)/float64(len(s)) > 0.3 {
		return true
	}

	for i := 0; i < len(s)-2; i++ {
		if s[i] == s[i+1] && s[i] == s[i+2] {
			return true
		}
	}

	upperCount := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			upperCount++
		}
	}
	if float64(upperCount)/float64(len(s)) > 0.5 && len(s) > 3 {
		return true
	}

	return false
}
