package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"time"

	"github.com/dreampuf/evernote-sdk-golang/client"
	"github.com/dreampuf/evernote-sdk-golang/notestore"
	"github.com/dreampuf/evernote-sdk-golang/types"
)

func main() {
	cfg, err := ReadConfig()
	if err != nil {
		log.Fatal(err)
	}
	c := newClient(cfg)
	posts := c.GetPostList()
	if changed := checkMeta(cfg, posts); !changed {
		log.Println("remote posts equal with meta json")
		return
	}
	log.Println("start to generate htmls")
	writeContent("", "changed", "data", "true")
	buf, _ := json.Marshal(posts)
	dir := cfg.ReleaseDir
	writeContent(dir, "meta", "json", string(buf))
	for _, post := range posts {
		log.Println(post)
		content, err := c.FetchContent(post.GUID)
		if err != nil {
			log.Println(err)
			continue
		}
		err = writeContent(dir, post.Title, "html", content)
		if err != nil {
			log.Println(err)
		}
	}
	index := generateIndex(posts)
	writeContent(dir, "index", "html", index)
}

type Config struct {
	EvernoteToken   string `json:"evernote_token"`
	EvernoteGUID    string `json:"evernote_guid"`
	ReleaseDir      string `json:"release_dir"`
	ReleaseProject  string `json:"release_project"`
	ReleaseUserName string `json:"release_username"`
	ReleaseBranch   string `json:"release_branch"`
}

func ReadConfig() (*Config, error) {
	if ci := os.Getenv("CIRCLECI"); ci != "" {
		return readFromCircleCIEnv(), nil
	}
	return nil, errors.New("config not found")
}

func readFromCircleCIEnv() *Config {
	var cfg Config
	cfg.EvernoteToken = os.Getenv("TOKEN")
	cfg.EvernoteGUID = os.Getenv("GUID")
	cfg.ReleaseProject = os.Getenv("CIRCLE_PROJECT_REPONAME")
	cfg.ReleaseUserName = os.Getenv("CIRCLE_PROJECT_USERNAME")
	cfg.ReleaseBranch = os.Getenv("RELEASE_BRANCH")
	cfg.ReleaseDir = "public"
	return &cfg
}

type Post struct {
	GUID    string `json:"guid"`
	Title   string `json:"title"`
	Update  int64  `json:"update"`
	Content string `json:"-"`
}

func checkMeta(cfg *Config, posts map[string]Post) bool {
	project := cfg.ReleaseProject
	username := cfg.ReleaseUserName
	branch := cfg.ReleaseUserName
	metafile := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/meta.json", username, project, branch)
	resp, err := http.Get(metafile)
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println(resp.Status)
	if resp.StatusCode == 404 {
		return true
	}
	if resp.StatusCode != 200 {
		return false
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return true
	}
	var respM map[string]Post
	err = json.Unmarshal(buf, &respM)
	if err != nil {
		return true
	}
	if len(respM) != len(posts) {
		return true
	}
	for key, p := range posts {
		remoteP := respM[key]
		if p.Update != remoteP.Update {
			return true
		}
	}
	return false
}

func generateIndex(m map[string]Post) string {
	var posts []Post
	for _, p := range m {
		posts = append(posts, p)
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Update < posts[i].Update
	})
	var content string
	for _, p := range posts {
		link := fmt.Sprintf("<li><a href=\"%s.html\">%s</a></li>", p.Title, p.Title)
		content += link
	}
	content += fmt.Sprintf("last updated @%v", time.Now())
	return content
}

func writeContent(dir, title, ext, content string) error {
	os.MkdirAll(dir, 0755)
	p := path.Join(dir, title+"."+ext)
	return ioutil.WriteFile(p, []byte(content), 0755)
}

type Client struct {
	cfg    *Config
	token  string
	guid   string
	client *client.EvernoteClient
}

func newClient(cfg *Config) *Client {
	c := client.NewClient("", "", client.YINXIANG)
	cc := &Client{
		cfg:    cfg,
		token:  cfg.EvernoteToken,
		guid:   cfg.EvernoteGUID,
		client: c,
	}
	return cc
}

func (c *Client) GetPostList() map[string]Post {
	store, err := c.client.GetNoteStore(c.token)
	if err != nil {
		log.Fatal(err)
	}
	bloguuid := types.GUID(c.guid)
	filter := notestore.NoteFilter{
		NotebookGuid: &bloguuid,
	}
	t := true
	resSpec := notestore.NotesMetadataResultSpec{
		IncludeTitle:   &t,
		IncludeUpdated: &t,
	}
	ll, err := store.FindNotesMetadata(c.token, &filter, 0, 100, &resSpec)
	if err != nil {
		log.Fatal(err)
	}
	res := make(map[string]Post)
	notes := ll.GetNotes()
	for _, note := range notes {
		p := Post{
			GUID:   string(note.GUID),
			Title:  string(note.GetTitle()),
			Update: int64(note.GetUpdated()),
		}
		res[string(note.GUID)] = p
	}
	return res
}

func (c *Client) FetchContent(guid string) (string, error) {
	store, err := c.client.GetNoteStore(c.token)
	if err != nil {
		return "", err
	}
	noteguid := types.GUID(guid)
	r, err := store.GetNote(c.token, noteguid, true, true, false, false)
	if err != nil {
		return "", err
	}
	return r.GetContent(), nil
}
