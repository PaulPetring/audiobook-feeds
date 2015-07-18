package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/honky/feeds"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type handler func(w http.ResponseWriter, r *http.Request)

var file_list EnhancedFileInfos
var config *AudioFeedConfig
var tmp *template.Template

type AudioFeedConfig struct {
	Feed_name         string
	Feed_webUrl       string
	Feed_port         int
	Feed_webRoot      string
	Feed_description  string
	Feed_author       string
	Feed_author_email string
	Feed_feeds_dir    string
	Feed_files_dir    string
	Feed_folders_dir  string
	Feed_theme        string
	Feed_enableAuth   bool
	Feed_username     string
	Feed_password     string
}

//reads config.json values and creates one from config.default.json in case there is none
//is available in templates
func GetConfig() (retConfig *AudioFeedConfig, err error) {

	if config != nil {
		retConfig = config
	}
	retConfig = &AudioFeedConfig{}
	confPath := getDirNameOfOperation() + "/config.json"
	dat, err := ioutil.ReadFile(confPath)

	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			dat, err = ioutil.ReadFile(getDirNameOfOperation() + "/config.default.json")
			if err != nil {
				return
			}
			err = ioutil.WriteFile(confPath, dat, os.FileMode(0664)) // https://golang.org/pkg/os/#FileMode
			if err != nil {
				return
			}
		} else {
			fmt.Println("Error : Could not read config path %s : ", confPath)
			return
		}
	}

	fmt.Println(string(dat))

	if err := json.Unmarshal(dat, &retConfig); err != nil {
		panic(err)
	}
	fmt.Println(retConfig)
	config = retConfig
	return
}

func BasicAuth(pass handler) handler {

	return func(w http.ResponseWriter, r *http.Request) {

		if config.Feed_enableAuth == false {
			return
		}

		if len(r.Header["Authorization"]) == 0 {
			//http.Error(w, "authorization failed", http.StatusUnauthorized
			w.Header().Set("WWW-Authenticate", "Basic realm=\"private\"")
			w.WriteHeader(401)
			w.Write([]byte("401 Unauthorized\n"))
			return
		}

		auth := strings.SplitN(r.Header["Authorization"][0], " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			http.Error(w, "bad syntax", http.StatusBadRequest)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !Validate(pair[0], pair[1]) {
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}

		pass(w, r)
	}
}

func Validate(username, password string) bool {
	if config.Feed_enableAuth == false {
		return true
	} else {
		if username == config.Feed_username && password == config.Feed_password {
			return true
		}
	}
	return false
}

type EnhancedFileInfos []EnhancedFileInfo

//part of the sorting interface EnhancedFileInfos
func (slice EnhancedFileInfos) Len() int {
	return len(slice)
}

//part of the sorting interface EnhancedFileInfos
func (slice EnhancedFileInfos) Less(i, j int) bool {
	//you would expect something like this in case of sorting by name
	//return slice[i].Name() < slice[j].Name()
	//but this one is better when sorting over multiple folders etc
	return slice[i].FullPath < slice[j].FullPath
}

//part of the sorting interface EnhancedFileInfos
func (slice EnhancedFileInfos) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type foldersData struct {
	Files  EnhancedFileInfos
	Parent string
}

//checks a FileInfo if it is a video or audio file
func isMediaFile(f os.FileInfo) bool {
	if isAudioFile(f) {
		return true
	}
	if isVideoFile(f) {
		return true
	}
	return false
}

//checks a FileInfo if it is a video file
func isVideoFile(f os.FileInfo) bool {
	if strings.HasSuffix(strings.ToLower(f.Name()), ".mpg") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(f.Name()), ".avi") {
		return true
	}
	return false
}

//checks a FileInfo if it is a audio file
func isAudioFile(f os.FileInfo) bool {
	if strings.HasSuffix(strings.ToLower(f.Name()), ".mp3") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(f.Name()), ".mp4") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(f.Name()), ".m4a") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(f.Name()), ".ogg") {
		return true
	}
	return false
}

//encodes an URL
//is available in templates
func encode_url(str string) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

//walking function for filepath
func visit(fullPath string, fi os.FileInfo, err error) error {
	if isMediaFile(fi) {
		file_list = append(file_list, EnhancedFileInfo{fi, path.Dir(fullPath), fullPath, isAudioFile(fi), isVideoFile(fi), isMediaFile(fi), fi.IsDir()})
	}
	return nil
}

//creates a feed by walking all its files including sub folders
func createFeedFromFolder(folder_path string, feedType string) (feed_content string, err error) {
	//all files that you want to server
	walk_path := strings.Replace(folder_path, config.Feed_webRoot+config.Feed_feeds_dir+"/"+feedType+"", "", 1)

	fmt.Println("CreateThread for %s in %s", folder_path, feedType)
	fmt.Println("CreateThread walk_path: ", walk_path)

	//check if path is existing before walking it
	if _, err := os.Stat(walk_path); err == nil {

		//read files
		file_list = file_list[:0]
		err = filepath.Walk(walk_path, visit)
		sort.Sort(file_list)

		//create feed
		now := time.Now()
		feed := &feeds.Feed{
			Title:       config.Feed_name,
			Link:        &feeds.Link{Href: config.Feed_webUrl},
			Description: config.Feed_description,
			Author:      &feeds.Author{config.Feed_author, config.Feed_author_email},
			Created:     now,
		}

		for _, file_path := range file_list {
			file_stat, err := os.Stat(file_path.FullPath)
			if err != nil {
				continue
			}

			parsed_url, err := encode_url(file_path.FullPath)
			if err != nil {
				return "", err
			}

			feed.Items = append(feed.Items, &feeds.Item{
				Title: file_stat.Name(),
				//TODO: improve encoding of urls
				Link:        &feeds.Link{Href: config.Feed_webUrl + config.Feed_webRoot + parsed_url, Length: file_stat.Size(), Type: "audio/mpeg"},
				Description: file_stat.Name() + "(" + humanize.Bytes(uint64(file_stat.Size())) + ") " + file_path.FullPath,
				Created:     file_stat.ModTime(),
			})
		}

		if feedType == "atom" {
			feed_content, err = feed.ToAtom()
		} else {
			feed_content, err = feed.ToRss()
		}
	}
	return
}

//creates a rss feed from a request
func rss_handler(w http.ResponseWriter, r *http.Request) {
	rss_feed_content, _ := createFeedFromFolder(config.Feed_files_dir+"/"+strings.Replace(r.URL.Path[1:], config.Feed_webRoot+config.Feed_feeds_dir+"/rss/", " ", -1), "rss")
	fmt.Fprintf(w, "%s", rss_feed_content)
}

//creates a rss feed from a request
func atom_handler(w http.ResponseWriter, r *http.Request) {
	atom_feed_content, _ := createFeedFromFolder(config.Feed_files_dir+"/"+strings.Replace(r.URL.Path[1:], config.Feed_webRoot+config.Feed_feeds_dir+"/atom/", " ", -1), "atom")
	fmt.Fprintf(w, "%s", atom_feed_content)
}

type EnhancedFileInfo struct {
	os.FileInfo
	Dir         string
	FullPath    string
	IsAudioFile bool
	IsVideoFile bool
	IsMediaFile bool
	IsDir       bool
}

func folders_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	fmt.Println(len(config.Feed_webRoot))
	fmt.Println(len(config.Feed_folders_dir))
	fmt.Println(len(config.Feed_webRoot) + len(config.Feed_folders_dir))
	fmt.Println(len(config.Feed_webRoot + config.Feed_folders_dir))
	cleanURLPath := r.URL.Path[(len(config.Feed_webRoot+config.Feed_folders_dir) + 0):]

	fileRoot := http.Dir(getDirNameOfOperation() + "/" + config.Feed_files_dir)
	fmt.Println("URL PATH: %s", r.URL.Path)
	fmt.Println("Clean URL PATH: %s", cleanURLPath)

	var err error
	if tmp == nil {
		tmplTxt, err := ioutil.ReadFile(getDirNameOfOperation() + "/themes/" + config.Feed_theme + "/foldersTemplate.html")
		if err != nil {
			fmt.Println("Error reading template file: ", err)
			http.Error(w, "Error reading template file", 500)
			return
		}
		x := template.New("FolderPage")
		funcs := template.FuncMap{"GetConfig": GetConfig, "Encode_url": encode_url}
		tmp = template.Must(x.Funcs(funcs).Parse(string(tmplTxt)))

	}
	if err != nil {
		fmt.Println("Error parsing foldersTemplate.html:")
		http.Error(w, "Error parsing foldersTemplate.html", 500)
		return
	}

	fpd := foldersData{Files: make([]EnhancedFileInfo, 0), Parent: cleanURLPath}
	dir, err := fileRoot.Open(cleanURLPath)

	if err != nil {
		fmt.Println("Error scanning files folder: ", err)
		if strings.Contains(err.Error(), "no such file") {
			http.Error(w, "File/Folder not found.", 404)
		} else {
			http.Error(w, "Unknown error scanning files folder", 500)
		}
		return
	}

	defer dir.Close()
	fis, err := dir.Readdir(-1)

	if err != nil {
		if strings.Contains(err.Error(), "not a directory") {
			http.Redirect(w, r, config.Feed_webRoot+config.Feed_files_dir+"/"+cleanURLPath, 307)
		} else {
			http.Error(w, "Unknown error scanning files folder", 500)
		}
		return
	}

	for _, fi := range fis {
		if strings.HasPrefix(fi.Name(), ".") { //skip invisible files
			continue
		}
		fpd.Files = append(fpd.Files, EnhancedFileInfo{fi, cleanURLPath, cleanURLPath + fi.Name(), isAudioFile(fi), isVideoFile(fi), isMediaFile(fi), fi.IsDir()})
	}

	sort.Sort(fpd.Files)

	err = tmp.Execute(w, fpd)
	if err != nil {
		fmt.Println("Error executing template: ", err)
		http.Error(w, "Error executing template", 500)
	}
}

func getFileNameOfOperation() string {
	//pragmatically find local running path
	_, filename, _, _ := runtime.Caller(1)
	return filename
}
func getDirNameOfOperation() string {
	return path.Dir(getFileNameOfOperation())
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "404 File/Folder not found: "+r.URL.Path)
}

//gogogo
func main() {

	var err error
	config, err = GetConfig()
	if err != nil {
		fmt.Println("Error in GetConfig : ", err)
		return
	}

	fmt.Println("Starting Server: " + getDirNameOfOperation() + "/" + config.Feed_files_dir + "/")

	//create local static file server
	fs := http.StripPrefix(config.Feed_webRoot+config.Feed_files_dir+"/", http.FileServer(http.Dir(getDirNameOfOperation()+"/"+config.Feed_files_dir+"/")))
	http.HandleFunc(config.Feed_webRoot+config.Feed_files_dir+"/", BasicAuth(fs.ServeHTTP))

	http.HandleFunc("/", NotFoundHandler)

	//enable folder handler to offer a small interface
	http.HandleFunc(config.Feed_webRoot+config.Feed_folders_dir+"/", BasicAuth(folders_handler))

	//create feeds by type
	http.HandleFunc(config.Feed_webRoot+config.Feed_feeds_dir+"/rss/", BasicAuth(rss_handler))
	http.HandleFunc(config.Feed_webRoot+config.Feed_feeds_dir+"/atom/", BasicAuth(atom_handler))

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Feed_port), nil)) //
}
