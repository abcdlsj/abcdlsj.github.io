package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	htmltmpl "html/template"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
	PubURL      string   `toml:"pub_url"`
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
				ThemeID: &d2themescatalog.NeutralDefault.ID,
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
			t, err := parsePostDate(s)
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
		"formatChangelog": func(s string) htmltmpl.HTML {
			if s == "" {
				return ""
			}
			lines := strings.Split(strings.TrimSpace(s), "\n")
			var result strings.Builder
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				result.WriteString(line)
				result.WriteString("<br>")
			}
			return htmltmpl.HTML(result.String())
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
	Title         string   `yaml:"title"`
	Date          string   `yaml:"date"`
	Tags          []string `yaml:"tags"`
	Hide          bool     `yaml:"hide"`
	Menus         []string `yaml:"menus"`
	Wip           bool     `yaml:"wip"`
	HideToc       bool     `yaml:"hideToc"`
	Hero          string   `yaml:"hero"`
	Thumbnail     string   `yaml:"thumbnail"`
	Cover         string   `yaml:"cover"`
	Description   string   `yaml:"description"`
	Summary       string   `yaml:"summary"`
	Languages     []string `yaml:"languages"`
	Changelog     string   `yaml:"changelog"`
	Sidebar       string   `yaml:"sidebar"`
	TimelineTitle string   `yaml:"timeline_title"`
}

func unmarshalPostMeta(meta map[string]interface{}) PostMeta {
	title := pickString(meta, "title")
	date := pickString(meta, "date")
	tags := pickStringSlice(meta, "tags")
	menus := pickStringSlice(meta, "menus")
	wip := pickBool(meta, "wip", "draft")
	hideToc := pickBool(meta, "hideToc")

	// New schema prefers `cover`; keep backward compatibility with `thumbnail` and `hero`.
	cover := pickString(meta, "cover", "thumbnail", "hero")
	thumbnail := pickString(meta, "thumbnail")
	hero := pickString(meta, "hero")
	if thumbnail == "" {
		thumbnail = cover
	}

	description := pickString(meta, "description", "summary", "excerpt")
	summary := pickString(meta, "summary", "description", "excerpt")

	langs := pickStringSlice(meta, "languages", "language")
	if len(langs) == 0 {
		langs = []string{"en"}
	}
	sidebar := strings.ToLower(strings.TrimSpace(pickString(meta, "sidebar")))
	if sidebar == "" {
		sidebar = "toc"
	}
	if sidebar != "timeline" {
		sidebar = "toc"
	}

	hide := pickBool(meta, "hide")
	// `published: false` => hidden post.
	if published, ok := pickOptionalBool(meta, "published"); ok {
		hide = !published
	}
	// `toc: false` => hide toc.
	if toc, ok := pickOptionalBool(meta, "toc"); ok {
		hideToc = !toc
	}

	if title == "" {
		title = "Untitled"
	}
	if date == "" {
		date = "1970-01-01T00:00:00+00:00"
	}

	return PostMeta{
		Title:         title,
		Date:          date,
		Tags:          tags,
		Hide:          hide,
		Menus:         menus,
		Wip:           wip,
		HideToc:       hideToc,
		Hero:          hero,
		Thumbnail:     thumbnail,
		Cover:         cover,
		Description:   description,
		Summary:       summary,
		Languages:     langs,
		Changelog:     pickString(meta, "changelog"),
		Sidebar:       sidebar,
		TimelineTitle: pickString(meta, "timeline_title"),
	}
}

type TimelineItem struct {
	ID    string
	Date  string
	Title string
}

type Post struct {
	Site       CfgVar
	Meta       PostMeta
	Body       string
	Uname      string
	TocContent string
	MDData     string
	Cover      string
	Excerpt    string
	Timeline   []TimelineItem
}

type Tag struct {
	Site   CfgVar
	Name   string
	Refers []Refer
}

type Refer struct {
	Title   string
	Uname   string
	Meta    PostMeta
	Cover   string
	Excerpt string
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
		Title:   post.Meta.Title,
		Uname:   post.Uname,
		Meta:    post.Meta,
		Cover:   post.Cover,
		Excerpt: post.Excerpt,
	}

	for _, tag := range tagNames {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
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
	rawMD := string(data)
	processedMD, timelineItems := extractTimeline(rawMD)

	var buf bytes.Buffer
	context := parser.NewContext()
	if err := md.Convert([]byte(processedMD), &buf, parser.WithContext(context)); err != nil {
		return Post{}, fmt.Errorf("failed to convert markdown, file: %s, err: %w", cleanName, err)
	}

	postMeta := unmarshalPostMeta(meta.Get(context))

	post := Post{
		Site:     cfgVar,
		Meta:     postMeta,
		Body:     applyTimelineSegments(buf.String(), timelineItems),
		Uname:    generateUniqueURL(cleanName),
		MDData:   rawMD,
		Timeline: timelineItems,
	}

	enrichPost(&post)

	if !post.Meta.HideToc && post.Meta.Sidebar != "timeline" {
		tocData := []byte(processedMD)
		doc := tocMd.Parser().Parse(text.NewReader(tocData))

		tree, err := toc.Inspect(doc, tocData)
		if err != nil {
			return Post{}, fmt.Errorf("failed to inspect toc, file: %s, err: %w", cleanName, err)
		}

		var tocBuf bytes.Buffer

		if len(tree.Items) != 0 {
			treeList := toc.RenderList(tree)

			if err := tocMd.Renderer().Render(&tocBuf, tocData, treeList); err != nil {
				return Post{}, fmt.Errorf("failed to render toc, file: %s, err: %w", cleanName, err)
			}
		}

		post.TocContent = strings.TrimSuffix(strings.TrimPrefix(tocBuf.String(), "<ul>"), "</ul>")
		post.TocContent = strings.ReplaceAll(post.TocContent, "Table of Contents", "")
	}

	return post, nil
}

func enrichPost(post *Post) {
	post.Cover = detectPostCover(*post)
	post.Excerpt = buildPostExcerpt(*post)
}

var (
	mdImgRe        = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+(?:\s+"[^"]*")?)\)`)
	htmlImgRe      = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
	timelineLineRe = regexp.MustCompile(`^\[timeline:\s*([^|\]]+)\s*\|\s*([^\]]+)\]\s*$`)
	mdLinkRe       = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	htmlTagRe      = regexp.MustCompile(`<[^>]+>`)
	spaceRe        = regexp.MustCompile(`\s+`)
	nonAlnumRe     = regexp.MustCompile(`[^a-z0-9]+`)
	decoration     = strings.NewReplacer("`", "", "*", "", "_", "", "#", "", ">", "", "-", "", "!", "")
)

func extractTimeline(rawMD string) (string, []TimelineItem) {
	lines := strings.Split(rawMD, "\n")
	var out []string
	items := make([]TimelineItem, 0)
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			out = append(out, line)
			continue
		}

		if !inCodeBlock {
			matches := timelineLineRe.FindStringSubmatch(trimmed)
			if len(matches) == 3 {
				date := strings.TrimSpace(matches[1])
				title := strings.TrimSpace(matches[2])
				id := timelineAnchorID(len(items)+1, date, title)

				out = append(out, fmt.Sprintf(`<!-- timeline-marker:%s -->`, id))
				items = append(items, TimelineItem{
					ID:    id,
					Date:  date,
					Title: title,
				})
				continue
			}
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n"), items
}

func applyTimelineSegments(body string, items []TimelineItem) string {
	if len(items) == 0 {
		return body
	}

	markerRe := regexp.MustCompile(`<!--\s*timeline-marker:([a-z0-9\-]+)\s*-->`)
	matches := markerRe.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return body
	}

	var b strings.Builder
	b.WriteString(body[:matches[0][0]])

	for i, match := range matches {
		id := body[match[2]:match[3]]
		segmentStart := match[1]
		segmentEnd := len(body)
		if i+1 < len(matches) {
			segmentEnd = matches[i+1][0]
		}

		b.WriteString(fmt.Sprintf(`<div id="%s" class="timeline-anchor" data-timeline-anchor></div>`, id))
		b.WriteString(fmt.Sprintf(`<div class="timeline-segment" data-timeline-segment data-timeline-id="%s">`, id))
		b.WriteString(body[segmentStart:segmentEnd])
		b.WriteString(`</div>`)
	}

	return b.String()
}

func timelineAnchorID(index int, date, title string) string {
	base := strings.ToLower(strings.TrimSpace(date + "-" + title))
	base = nonAlnumRe.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "timeline"
	}
	return fmt.Sprintf("tl-%d-%s", index, base)
}

func detectPostCover(post Post) string {
	if post.Meta.Cover != "" {
		return strings.TrimSpace(post.Meta.Cover)
	}
	if post.Meta.Thumbnail != "" {
		return strings.TrimSpace(post.Meta.Thumbnail)
	}
	if post.Meta.Hero != "" {
		return strings.TrimSpace(post.Meta.Hero)
	}

	if matches := mdImgRe.FindStringSubmatch(post.MDData); len(matches) > 1 {
		candidate := strings.TrimSpace(matches[1])
		if idx := strings.Index(candidate, " "); idx > 0 {
			candidate = candidate[:idx]
		}
		candidate = strings.Trim(candidate, `<>`)
		if candidate != "" {
			return candidate
		}
	}

	if matches := htmlImgRe.FindStringSubmatch(post.Body); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

func buildPostExcerpt(post Post) string {
	if post.Meta.Description != "" {
		return post.Meta.Description
	}
	if post.Meta.Summary != "" {
		return post.Meta.Summary
	}

	content := stripFrontMatter(post.MDData)
	lines := strings.Split(content, "\n")
	var b strings.Builder
	inCodeBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "![") {
			continue
		}

		b.WriteString(trimmed)
		b.WriteString(" ")
		if b.Len() > 500 {
			break
		}
	}

	excerpt := b.String()
	excerpt = mdLinkRe.ReplaceAllString(excerpt, `$1`)
	excerpt = htmlTagRe.ReplaceAllString(excerpt, " ")
	excerpt = decoration.Replace(excerpt)
	excerpt = strings.TrimSpace(spaceRe.ReplaceAllString(excerpt, " "))

	if excerpt == "" {
		return ""
	}

	rs := []rune(excerpt)
	if len(rs) > 180 {
		return string(rs[:180]) + "..."
	}
	return excerpt
}

func stripFrontMatter(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n")
		}
	}

	return content
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
		pubDate, err := parsePostDate(post.Meta.Date)
		if err != nil {
			pubDate = time.Unix(0, 0).UTC()
		}
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
	rssFile.WriteString(`<?xml-stylesheet href="/static/pretty_feed.xsl" type="text/xsl"?>` + "\n")
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

func generateCNAME() error {
	domain := strings.TrimPrefix(cfgVar.PubURL, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	return os.WriteFile(path.Join(cfgVar.Build.Output, "CNAME"), []byte(domain), 0644)
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

	if err := generateCNAME(); err != nil {
		log.Fatal(err)
	}

	fmt.Println(cr.PLCyan("All done!!!"))
}

func dateCompare(a, b string) bool {
	ta, err := parsePostDate(a)
	if err != nil {
		log.Fatal(err)
		return false
	}
	tb, err := parsePostDate(b)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return ta.After(tb)
}

func parsePostDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty date")
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}

	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("unsupported date format: %q, err: %w", value, lastErr)
}

func pickString(meta map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		val, ok := meta[key]
		if !ok || val == nil {
			continue
		}

		switch v := val.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case fmt.Stringer:
			s := strings.TrimSpace(v.String())
			if s != "" {
				return s
			}
		default:
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func pickStringSlice(meta map[string]interface{}, keys ...string) []string {
	for _, key := range keys {
		val, ok := meta[key]
		if !ok || val == nil {
			continue
		}
		switch v := val.(type) {
		case []string:
			return sanitizeStringSlice(v)
		case []interface{}:
			out := make([]string, 0, len(v))
			for _, item := range v {
				out = append(out, strings.TrimSpace(fmt.Sprintf("%v", item)))
			}
			return sanitizeStringSlice(out)
		case string:
			if strings.TrimSpace(v) == "" {
				return nil
			}
			parts := strings.Split(v, ",")
			return sanitizeStringSlice(parts)
		}
	}
	return nil
}

func sanitizeStringSlice(vs []string) []string {
	out := make([]string, 0, len(vs))
	for _, item := range vs {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func pickBool(meta map[string]interface{}, keys ...string) bool {
	val, ok := pickOptionalBool(meta, keys...)
	return ok && val
}

func pickOptionalBool(meta map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		val, ok := meta[key]
		if !ok || val == nil {
			continue
		}

		switch v := val.(type) {
		case bool:
			return v, true
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "yes", "1", "on":
				return true, true
			case "false", "no", "0", "off":
				return false, true
			}
		case int:
			return v != 0, true
		case int64:
			return v != 0, true
		case float64:
			return v != 0, true
		}
	}
	return false, false
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
