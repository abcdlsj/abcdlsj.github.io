---
title: "Pasty - Beautiful pastebin website with Go, Turnstile, OAuth"
date: 2023-12-08T22:47:37+08:00
tags:
  - Weekly
  - OAuth
  - Turnstile
hide: false
---

## Background
I had try to build many of `Go` simple site, Like `Golink`, `GProbe` and so on.
But it's not secure enough, and very easy to be attacked.
At these days, I want to build a `Pastebin` website and its need will powerful **Security**.

## Let us build Pasty

Its all old way like I posted before. 
- `Go` serve HTTP endpoints.
- template to render HTML.
- SQLite3 to store and query data.

### Endpoint design
For simply, I just make 2 endpoints for `Pasty`:
1. `/`: to get index page(`GET`) and create a new paste(`POST`)
2. `/paste/`: to get paste page(`GET`) and delete it(`DELETE`)

This is the whole APIs I need to write.

Let's start:
```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		pastes := getAllPastes(db)
		tmpl.ExecuteTemplate(w, "index.html", pastes)
	} else {
		r.ParseForm()
		content := r.Form.Get("content")
		insertPaste(db, escapeContent(content))
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
})

http.HandleFunc("/paste/", func(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Path[len("/paste/"):]

	if r.Method == "GET" {
		paste := getPasteWithID(db, uid)
		if paste.isNil() {
			http.NotFound(w, r)
			return
		}
		tmpl.ExecuteTemplate(w, "paste.html", paste)
	} else {
		deletePaste(db, uid)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
})
```

Then we need to write `tmpl` and implement `CRUD`(`getAllPastes`, `getPasteWithID`, `insertPaste`, `deletePaste`)

### ORM or Raw-SQL
Actually, I'm a original-thinker, I don't like to use third-party packages.
But `ORM` will make program easier.

I use `SQLite3` and `Gorm` to store and query data.
`Gorm` is easy to use, it provide a lot of functionality.

```go
type Paste struct {
	ID        int       `gorm:"column:id"`
	UID       string    `gorm:"column:uid"`
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func getAllPastes(db *gorm.DB) []Paste {
	pastes := []Paste{}
	db.Find(&pastes)
	return pastes
}

func getPasteWithID(db *gorm.DB, uid string) Paste {
	paste := Paste{}
	db.First(&paste, "uid = ?", uid)
	return paste
}

func insertPaste(db *gorm.DB, content string) {
	paste := Paste{UID: uuid.New().String(), Content: content, CreatedAt: time.Now()}
	db.Create(&paste)
}

func deletePaste(db *gorm.DB, uid string) {
	db.Delete(&Paste{}, "uid = ?", uid)
}
```

There are all we need!

Oh.. maybe we need to init a database?
Right, It's also very easy to init databse, `Gorm` will help you.
```go
func initDB(filepath string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(filepath), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = db.AutoMigrate(&Paste{})
	if err != nil {
		log.Fatal(err)
	}

	return db
}
```
We just need to open the database and migrate the table. If table not exist, it will create it. and if exist, it will do nothing.

### templates
I thought `Template` is very powerful when build simple website.
We just need to create 2 pages: `index.html` and `paste.html`.

```html
<!DOCTYPE html>
<html>
<head>
    <title>Pasty</title>
</head>
<body>
    <h1>Pasty</h1>
    <p>Input your paste</p>
    <form method="POST">
        <textarea name="content" required rows="30"></textarea>
        <br/>
        <input type="submit" value="Submit">
    </form>
    <hr/>
    <h2>Pasted</h2>
    {{range .}}
        <div>
            <a href="/paste/{{.UID}}">{{.UID}}</a>
            <p><pre><code>{{truncate .Content 200}}</code></pre></p>
        </div>
    {{end}}
</body>
</html>
```

```html
<!DOCTYPE html>
<html>

<head>
    <title>Paste</title>
</head>

<body>
    <h1>Paste</h1>
    <a href="/">Home</a>
    <div>
        <h2>{{.UID}}</h2>
        <pre><code>{{.Content}}</code></pre>
        <p>{{.CreatedAt}}</p>
        <form method="POST">
            <input type="submit" value="Delete">
        </form>
    </div>
</body>

</html>
```

`CSS` is the hardest part I thoguht when I write any website. So I use `water.css` for styling.
> `Water.css` is a drop-in collection of CSS styles to make simple websites like this just a little bit nicer.<https://watercss.kognise.dev/>

You just need to import `water.css` to `html` file, and you will got a pretty website with auto dark mode.
Add this to `head` element.
```
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/water.css@2/out/water.css">
```

### Run it!

<img src="/static/img/pasty-1.png" width="800">

Looks pretty good!
> It's use `gg font` as `font-family`.

At the last step, we will to add `Turnstile`.

## Turnstile
> Turnstile is Cloudflare’s smart CAPTCHA alternative. It can be embedded into any website without sending traffic through Cloudflare and works without showing visitors a CAPTCHA. [Cloudflare](https://developers.cloudflare.com/turnstile/get-started)


COMMING SOON