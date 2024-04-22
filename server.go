package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/andreimarcu/linx-server/auth/apikeys"
	"github.com/andreimarcu/linx-server/backends"
	"github.com/andreimarcu/linx-server/backends/localfs"
	"github.com/andreimarcu/linx-server/cleanup"
	"github.com/flosch/pongo2"
	"github.com/vharitonsky/iniflags"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

type headerList []string

func (h *headerList) String() string {
	return strings.Join(*h, ",")
}

func (h *headerList) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var Config struct {
	bind                   string
	filesDir               string
	metaDir                string
	siteName               string
	siteURL                string
	sitePath               string
	selifPath              string
	certFile               string
	keyFile                string
	disableSecurityHeaders bool
	maxSize                int64
	maxExpiry              uint64
	defaultExpiry          uint64
	realIp                 bool
	noLogs                 bool
	allowHotlink           bool
	basicAuth              bool
	authFile               string
	addHeaders             headerList
	noDirectAgents         bool
	forceRandomFilename    bool
	accessKeyCookieExpiry  uint64
	customPagesDir         string
	cleanupEveryMinutes    uint64
	extraFooterText        string
	maxDurationTime        uint64
	maxDurationSize        int64
	disableAccessKey       bool
	defaultRandomFilename  bool
}

var Templates = make(map[string]*pongo2.Template)
var TemplateSet *pongo2.TemplateSet
var staticBox *rice.Box
var timeStarted time.Time
var timeStartedStr string
var storageBackend backends.StorageBackend
var customPages = make(map[string]string)
var customPagesNames = make(map[string]string)

func setup() *web.Mux {
	mux := web.New()

	// middleware
	mux.Use(middleware.RequestID)

	if Config.realIp {
		mux.Use(middleware.RealIP)
	}

	if !Config.noLogs {
		mux.Use(middleware.Logger)
	}

	mux.Use(middleware.Recoverer)
	mux.Use(middleware.AutomaticOptions)
	if !Config.disableSecurityHeaders {
		mux.Use(ContentSecurityPolicy(defaultCSPOptions))
	}

	mux.Use(AddHeaders(Config.addHeaders))

	if Config.authFile != "" {
		mux.Use(apikeys.NewApiKeysMiddleware(apikeys.AuthOptions{
			AuthFile:      Config.authFile,
			UnauthMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
			BasicAuth:     Config.basicAuth,
			SiteName:      Config.siteName,
			SitePath:      Config.sitePath,
		}))
	}

	// make directories if needed
	err := os.MkdirAll(Config.filesDir, 0755)
	if err != nil {
		log.Fatal("Could not create files directory:", err)
	}

	err = os.MkdirAll(Config.metaDir, 0700)
	if err != nil {
		log.Fatal("Could not create metadata directory:", err)
	}

	if Config.siteURL != "" {
		// ensure siteURL ends wth '/'
		if lastChar := Config.siteURL[len(Config.siteURL)-1:]; lastChar != "/" {
			Config.siteURL = Config.siteURL + "/"
		}

		parsedUrl, err := url.Parse(Config.siteURL)
		if err != nil {
			log.Fatal("Could not parse siteurl:", err)
		}

		Config.sitePath = parsedUrl.Path
	} else {
		Config.sitePath = "/"
	}

	Config.selifPath = strings.TrimLeft(Config.selifPath, "/")
	if lastChar := Config.selifPath[len(Config.selifPath)-1:]; lastChar != "/" {
		Config.selifPath = Config.selifPath + "/"
	}

	storageBackend = localfs.NewLocalfsBackend(Config.metaDir, Config.filesDir)
	if Config.cleanupEveryMinutes > 0 {
		go cleanup.PeriodicCleanup(time.Duration(Config.cleanupEveryMinutes)*time.Minute, Config.filesDir, Config.metaDir, Config.noLogs)
	}

	// Template setup
	p2l, err := NewPongo2TemplatesLoader()
	if err != nil {
		log.Fatal("Error: could not load templates", err)
	}
	TemplateSet := pongo2.NewSet("templates", p2l)
	err = populateTemplatesMap(TemplateSet, Templates)
	if err != nil {
		log.Fatal("Error: could not load templates", err)
	}

	staticBox = rice.MustFindBox("static")
	timeStarted = time.Now()
	timeStartedStr = strconv.FormatInt(timeStarted.Unix(), 10)

	// Routing setup
	nameRe := regexp.MustCompile("^" + Config.sitePath + `(?P<name>[a-z0-9-\.]+)$`)
	selifRe := regexp.MustCompile("^" + Config.sitePath + Config.selifPath + `(?P<name>[a-z0-9-\.]+)$`)
	selifIndexRe := regexp.MustCompile("^" + Config.sitePath + Config.selifPath + `$`)

	if Config.authFile == "" || Config.basicAuth {
		mux.Get(Config.sitePath, indexHandler)
		mux.Get(Config.sitePath+"paste/", pasteHandler)
	} else {
		mux.Get(Config.sitePath, http.RedirectHandler(Config.sitePath+"API", 303))
		mux.Get(Config.sitePath+"paste/", http.RedirectHandler(Config.sitePath+"API/", 303))
	}
	mux.Get(Config.sitePath+"paste", http.RedirectHandler(Config.sitePath+"paste/", 301))

	mux.Get(Config.sitePath+"API/", apiDocHandler)
	mux.Get(Config.sitePath+"API", http.RedirectHandler(Config.sitePath+"API/", 301))

	mux.Post(Config.sitePath+"upload", uploadPostHandler)
	mux.Post(Config.sitePath+"upload/", uploadPostHandler)
	mux.Put(Config.sitePath+"upload", uploadPutHandler)
	mux.Put(Config.sitePath+"upload/", uploadPutHandler)
	mux.Put(Config.sitePath+"upload/:name", uploadPutHandler)

	mux.Delete(Config.sitePath+":name", deleteHandler)
	// Adding new delete path method to make linx-server usable with ShareX.
	mux.Get(Config.sitePath+"delete/:name", deleteHandler)

	mux.Get(Config.sitePath+"static/*", staticHandler)
	mux.Get(Config.sitePath+"favicon.ico", staticHandler)
	mux.Get(Config.sitePath+"robots.txt", staticHandler)
	mux.Get(nameRe, fileAccessHandler)
	mux.Post(nameRe, fileAccessHandler)
	mux.Get(selifRe, fileServeHandler)
	mux.Get(selifIndexRe, unauthorizedHandler)
	if Config.customPagesDir != "" {
		initializeCustomPages(Config.customPagesDir)
		for fileName := range customPagesNames {
			mux.Get(Config.sitePath+fileName, makeCustomPageHandler(fileName))
			mux.Get(Config.sitePath+fileName+"/", makeCustomPageHandler(fileName))
		}
	}

	mux.NotFound(notFoundHandler)

	return mux
}

func main() {
	flag.StringVar(&Config.bind, "bind", "127.0.0.1:8080",
		"host to bind to (default: 127.0.0.1:8080)")
	flag.StringVar(&Config.filesDir, "filespath", "files/",
		"path to files directory")
	flag.StringVar(&Config.metaDir, "metapath", "meta/",
		"path to metadata directory")
	flag.BoolVar(&Config.basicAuth, "basicauth", false,
		"allow logging by basic auth password")
	flag.BoolVar(&Config.noLogs, "nologs", false,
		"remove stdout output for each request")
	flag.BoolVar(&Config.allowHotlink, "allowhotlink", false,
		"Allow hotlinking of files")
	flag.StringVar(&Config.siteName, "sitename", "",
		"name of the site")
	flag.StringVar(&Config.siteURL, "siteurl", "",
		"site base url (including trailing slash)")
	flag.StringVar(&Config.selifPath, "selifpath", "selif",
		"path relative to site base url where files are accessed directly")
	flag.Int64Var(&Config.maxSize, "maxsize", 4*1024*1024*1024,
		"maximum upload file size in bytes (default 4GB)")
	flag.Uint64Var(&Config.maxExpiry, "maxexpiry", 0,
		"maximum expiration time in seconds (default is 0, which is no expiry)")
	flag.Uint64Var(&Config.defaultExpiry, "default-expiry", 86400,
		"default expiration time in seconds (default is 86400, which is 1 day)")
	flag.StringVar(&Config.certFile, "certfile", "",
		"path to ssl certificate (for https)")
	flag.StringVar(&Config.keyFile, "keyfile", "",
		"path to ssl key (for https)")
	flag.BoolVar(&Config.realIp, "realip", false,
		"use X-Real-IP/X-Forwarded-For headers as original host")
	flag.StringVar(&Config.authFile, "authfile", "",
		"path to a file containing newline-separated scrypted auth keys")
	flag.Var(&Config.addHeaders, "addheader",
		"Add an arbitrary header to the response. This option can be used multiple times.")
	flag.BoolVar(&Config.noDirectAgents, "nodirectagents", false,
		"disable serving files directly for wget/curl user agents")
	flag.BoolVar(&Config.forceRandomFilename, "force-random-filename", false,
		"Force all uploads to use a random filename")
	flag.Uint64Var(&Config.accessKeyCookieExpiry, "access-cookie-expiry", 0, "Expiration time for access key cookies in seconds (set 0 to use session cookies)")
	flag.StringVar(&Config.customPagesDir, "custompagespath", "",
		"path to directory containing .md files to render as custom pages")
	flag.Uint64Var(&Config.cleanupEveryMinutes, "cleanup-every-minutes", 0,
		"How often to clean up expired files in minutes (default is 0, which means files will be cleaned up as they are accessed)")
	flag.StringVar(&Config.extraFooterText, "extra-footer-text", "",
		"Extra text above the footer for notices.")
	flag.Uint64Var(&Config.maxDurationTime, "max-duration-time", 0, "Time till expiry for files over max-duration-size. (Default is 0 for no-expiry.)")
	flag.Int64Var(&Config.maxDurationSize, "max-duration-size", 4*1024*1024*1024, "Size of file before max-duration-time is used to determine expiry max time. (Default is 4GB)")
	flag.BoolVar(&Config.disableAccessKey, "disable-access-key", false, "Disables access key usage. (Default is false.)")
	flag.BoolVar(&Config.defaultRandomFilename, "default-random-filename", true, "Makes it so the random filename is not default if set false. (Default is true.)")
	iniflags.Parse()

	mux := setup()

	if Config.certFile != "" {
		log.Printf("Serving over https, bound on %s", Config.bind)
		err := graceful.ListenAndServeTLS(Config.bind, Config.certFile, Config.keyFile, mux)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("Serving over http, bound on %s", Config.bind)
		err := graceful.ListenAndServe(Config.bind, mux)
		if err != nil {
			log.Fatal(err)
		}
	}
}
