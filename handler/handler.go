package handler

import (
	"errors"
	"github.com/grooveshark/golib/gslog"
	"github.com/kinghrothgar/goblin/conf"
	"github.com/kinghrothgar/goblin/storage/store"
	"github.com/kinghrothgar/goblin/templ"
	"net"
	"net/http"
	"regexp"
	"strings"
)

var (
	alphaReg            = regexp.MustCompile("^[A-Za-z]+$")
	browserUserAgentReg = regexp.MustCompile("Mozilla")
	textContentTypeReg  = regexp.MustCompile("^text/")
)

func getGobData(w http.ResponseWriter, r *http.Request) []byte {
	//parse the multipart form in the request
	err := r.ParseMultipartForm(11534336) // Random ass number?
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	//get a ref to the parsed multipart form
	m := r.MultipartForm
	str := m.Value["gob"][0]
	return []byte(str)
}

func validateUID(w http.ResponseWriter, uid string) error {
	// This is so someone can't access a horde goblin
	// by just puting the 'horde#uid' instead of 'horde/uid'
	// and prevents a lookup if it's obviously crap
	if len(uid) > conf.UIDLen || !alphaReg.MatchString(uid) {
		err := errors.New("invalid uid")
		gslog.Debug(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}

func validateHordeName(w http.ResponseWriter, hordeName string) {
	// TODO: horde max length?
	if len(hordeName) > 50 || !alphaReg.MatchString(hordeName) {
		gslog.Debug("invalid horde name")
		http.Error(w, "invalid horde name", http.StatusBadRequest)
	}
	return
}

// Request.RemoteAddress contains port, which we want to remove i.e.:
// "[::1]:58292" => "[::1]"
func ipAddrFromRemoteAddr(s string) string {
	host, _, _ := net.SplitHostPort(s)
	return net.ParseIP(host).String()
}

func getIpAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIp := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIp == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	// Why comment this out?
	//if hdrForwardedFor != "" {
	//        // X-Forwarded-For is potentially a list of addresses separated with ","
	//        parts := strings.Split(hdrForwardedFor, ",")
	//        for i, p := range parts {
	//                parts[i] = strings.TrimSpace(p)
	//        }
	//        // TODO: should return first non-local address
	//        return parts[0]
	//}
	return net.ParseIP(hdrRealIp).String()
}

func getPageType(r *http.Request) string {
	userAgent := r.Header.Get("User-Agent")
	params := r.URL.Query()
	_, cli := params["cli"]
	// If cli param present or Mozilla not found in the user agent, use plain text
	if cli || !browserUserAgentReg.MatchString(userAgent) {
		return "TEXT"
	}
	return "HTML"
}

func getLanguage(r *http.Request) string {
	params := r.URL.Query()
	// Firgure out language parameter
	lang := ""
	for key, _ := range params {
		// Skip the pat params that start with :
		if key[0] == ':' {
			continue
		}
		lang = strings.ToLower(key)
	}
	// Deal with aliases
	switch lang {
	case "markup":
	case "css":
	case "css.selector":
	case "clike":
	case "javascript", "js":
		lang = "javascript"
	case "java":
	case "php":
	case "coffeescript", "coffee":
		lang = "coffeescript"
	case "scss":
	case "bash", "sh":
		lang = "bash"
	case "c":
	case "cpp":
	case "python", "py":
		lang = "python"
	case "sql":
	case "groovy", "gvy", "gy", "gsh":
		lang = "groovy"
	case "http":
	case "ruby", "rb":
		lang = "ruby"
	case "gherkin":
	case "csharp":
	case "go":
	default:
		lang = ""
	}
	return lang
}

func GetRoot(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("GetRoot called")
	params := r.URL.Query()
	gslog.Debug("query params: %+v", params)
	pageType := getPageType(r)
	pageBytes, err := templ.GetHomePage(pageType)
	if err != nil {
		gslog.Debug("failed to write home with error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(pageBytes)
}

func GetGob(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("GetGob called")
	params := r.URL.Query()
	uid := params.Get(":uid")
	if err := validateUID(w, uid); err != nil {
		gslog.Debug(err.Error())
		// I think validateUID does this already. I would leave this logic here
		// and just have validateUID only take in uid and return an error
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data, _, err := store.GetGob(uid)
	if err != nil {
		gslog.Debug("failed to get gob with error: " + err.Error())
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// Firgure out language parameter
	lang := getLanguage(r)

	if lang == "" {
		gslog.Debug("GetGob writing data")
		w.Write(data)
		return
	}
	contentType := http.DetectContentType(data)
	if textContentTypeReg.MatchString(contentType) {
		data, err = templ.GetGobPage(lang, data)
		if err != nil {
			gslog.Debug("failed to get gob page with error: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Write(data)
}

func GetHorde(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("GetHorde called")
	params := r.URL.Query()
	hordeName := params.Get(":horde")
	horde, err := store.GetHorde(hordeName)
	if err != nil {
		gslog.Debug("failed to get horde with error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pageType := getPageType(r)
	pageBytes, err := templ.GetHordePage(pageType, hordeName, horde)
	if err != nil {
		gslog.Debug("failed to get horde with error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(pageBytes)
}

func PostGob(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("PostGob called")
	gslog.Debug("Request has header: %+v", r.Header)
	gslog.Debug("Request has host: %s", r.Host)
	gslog.Debug("Request has requestURI: %s", r.RequestURI)
	gobData := getGobData(w, r)
	uid := store.GetNewUID()
	ip := getIpAddress(r)
	gslog.Debug("uid: %s, ip: %s", uid, ip)
	if err := store.PutGob(uid, gobData, ip); err != nil {
		gslog.Debug("put gob failed with error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url := templ.BuildURL(uid)
	w.Write([]byte(url))
}

func PostHordeGob(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("PostHordeGob called")
	params := r.URL.Query()
	hordeName := params.Get(":horde")
	// Same thing as validateUID. I would have validateHordeName only take in
	// the name and return an error. Also, as of right now, if there was an
	// error how would execution stop?
	validateHordeName(w, hordeName)
	gobData := getGobData(w, r)
	uid := store.GetNewUID()
	ip := getIpAddress(r)
	gslog.Debug("uid: %s, ip: %s", uid, ip)
	if err := store.PutHordeGob(uid, hordeName, gobData, ip); err != nil {
		gslog.Debug("put horde gob failed with error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	url := templ.BuildURL(uid)
	w.Write([]byte(url))
}

func DelGob(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("DelGob called")
}

func DelHordeGob(w http.ResponseWriter, r *http.Request) {
	gslog.Debug("DelHordeGob called")
}
