package retvortaro

import (
	"bufio"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
)

type Page struct {
	ToEn, ToEo   map[string]string
	Translations []Translation
}

type Translation struct {
	From, To string
}

const tmpltFile = "t.tmplt"

var (
	page = Page{
		ToEn: make(map[string]string),
		ToEo: make(map[string]string),
	}
	tmplt *template.Template
)

func init() {
	load()
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/en/", enHandler)
	http.HandleFunc("/eo/", eoHandler)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := tmplt.ExecuteTemplate(w, tmpltFile, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func enHandler(w http.ResponseWriter, r *http.Request) {
	p := page
	word := fixX(path.Base(r.URL.Path))
	if word != "" {
		enWord := page.ToEn[strings.ToLower(word)]
		if enWord == "" {
			// Nothing found. Try normalizing the suffix and trying again.
			word0 := word
			word = fixEoSuffix(word)
			if enWord = page.ToEn[strings.ToLower(word)]; enWord == "" {
				word = word0
			}
		}		
		p.Translations = append(p.Translations, Translation{
			From: word,
			To:   enWord,
		})
	}
	if err := tmplt.ExecuteTemplate(w, tmpltFile, p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func eoHandler(w http.ResponseWriter, r *http.Request) {
	p := page
	word := path.Base(r.URL.Path)
	if word != "" {
		p.Translations = append(p.Translations, Translation{
			From: word,
			To:   page.ToEo[strings.ToLower(word)],
		})

		// Also look for the infinitive, in case this is a verb.
		enVerb := "to " + word
		eoVerb := page.ToEo[strings.ToLower(enVerb)]
		if eoVerb != "" {
			p.Translations = append(p.Translations, Translation{
				From: enVerb,
				To:   eoVerb,
			})
		}
	}
	if err := tmplt.ExecuteTemplate(w, tmpltFile, p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fixX(word string) string {
	for _, sub := range []struct{ from, to string }{
		{"cx", "ĉ"},
		{"Cx", "Ĉ"},
		{"gx", "ĝ"},
		{"Gx", "Ĝ"},
		{"hx", "ĥ"},
		{"Hx", "Ĥ"},
		{"jx", "ĵ"},
		{"Jx", "Ĵ"},
		{"sx", "ŝ"},
		{"Sx", "Ŝ"},
		{"ux", "ŭ"},
		{"Ux", "Ŭ"},
	} {
		word = strings.Replace(word, sub.from, sub.to, -1)
	}
	return word
}

func fixEoSuffix(word string) string {
	for _, suffix := range []struct{ from, to string }{
		{"as", "i"},
		{"is", "i"},
		{"os", "i"},
		{"us", "i"},
		{"u", "i"},
		{"jn", ""},
		{"n", ""},
		{"j", ""},
	} {
		if strings.HasSuffix(word, suffix.from) {
			word = strings.TrimSuffix(word, suffix.from) + suffix.to
			break
		}
	}
	return word
}

func load() {
	loadWords()
	loadTemplate()
}

func loadWords() {
	f, err := os.Open("espdic.txt")
	if err != nil {
		panic(err)
	}
	s := bufio.NewScanner(f)
	s.Scan() // Munch the first line. It's a comment.
	for s.Scan() {
		fs := strings.Split(s.Text(), ":")
		if len(fs) != 2 {
			continue
		}
		eo := strings.TrimSpace(fs[0])
		eoLow := strings.ToLower(eo)
		en := strings.TrimSpace(fs[1])

		page.ToEn[eoLow] = en

		for _, en := range strings.Split(en, ",") {
			en = strings.TrimSpace(en)
			enLow := strings.ToLower(en)
			if cur := page.ToEo[enLow]; len(cur) > 0 {
				page.ToEo[enLow] = cur + ", " + eo
			} else {
				page.ToEo[enLow] = eo
			}
		}
	}
	if err := s.Err(); err != nil {
		panic(err)
	}
}

func loadTemplate() {
	var err error
	tmplt, err = template.ParseFiles("t.tmplt")
	if err != nil {
		panic(err)
	}
}
