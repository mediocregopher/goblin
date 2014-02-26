package templ

import (
	"bytes"
	"errors"
	"github.com/grooveshark/golib/gslog"
	"github.com/kinghrothgar/goblin/storage"
	htmlTemplate "html/template"
	textTemplate "text/template"
)

type HordePage struct {
	Domain string
	Title  string
	Horde  storage.Horde
}

type HomePage struct {
	Domain string
	Title  string
}

type GobPage struct {
	Title    string
	Language string
	Data     string
}

var (
	htmlTemplates *htmlTemplate.Template
	textTemplates *textTemplate.Template
	domain        string
)

// Instead of contentType being a string I would have it be a const in this
// package. Makes it more clear why it's being used when you need use them in
// other packages
func executeTemplate(contentType string, templateName string, data interface{}) ([]byte, error) {
	var err error
	buf := &bytes.Buffer{}
	switch contentType {
	case "HTML":
		gslog.Debug("executeTemplate with contentType 'HTML' and template name '%s'", templateName)
		err = htmlTemplates.ExecuteTemplate(buf, templateName, data)
		break
	case "TEXT":
		gslog.Debug("executeTemplate with contentType 'TEXT' and template name '%s'", templateName)
		err = textTemplates.ExecuteTemplate(buf, templateName, data)
		break
	default:
		err = errors.New("invalid content type")
	}
	return buf.Bytes(), err
}

func unescaped(x string) interface{} {
	return htmlTemplate.HTML(x)
}

func Initialize(htmlTemplatesPath string, textTemplatesPath string, confDomain string) error {
	var err error
	htmlTemplates, err = htmlTemplate.ParseFiles(htmlTemplatesPath)
	if err != nil {
		return err
	}
	textTemplates, err = textTemplate.ParseFiles(textTemplatesPath)
	gslog.Debug("htmlTemplates: %+v", htmlTemplates)
	gslog.Debug("textTemplates: %+v", textTemplates)
	domain = confDomain
	return err
}

func GetHordePage(contentType string, hordeName string, horde storage.Horde) ([]byte, error) {
	p := &HordePage{Domain: domain, Title: "horde: " + hordeName, Horde: horde}
	return executeTemplate(contentType, "hordePage", p)
}

func GetGobPage(language string, data []byte) ([]byte, error) {
	p := &GobPage{Title: "gob: " + language + " syntax highlighted", Language: language, Data: string(data)}
	return executeTemplate("HTML", "gobPage", p)
}

func GetHomePage(contentType string) ([]byte, error) {
	p := &HomePage{Domain: domain, Title: "gobin: a cli pastebin"}
	return executeTemplate(contentType, "homePage", p)
}

func BuildURL(uid string) string {
	return "http://" + domain + "/" + uid + "\n"
}
