package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"

	"github.com/dreampuf/evernote-sdk-golang/client"
	"github.com/dreampuf/evernote-sdk-golang/notestore"
	"github.com/dreampuf/evernote-sdk-golang/types"
	"github.com/zhaojkun/yinxiangblog/utils"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cfg, err := ReadConfig()
	if err != nil {
		log.Fatal(err)
	}
	c := newClient(cfg)
	posts := c.GetPostList()
	if changed := c.CheckMeta(posts); !changed {
		log.Println("remote posts equal with meta json")
		return
	}
	log.Println("start to generate htmls")
	writeContent("", "changed", "data", "true")
	c.WriteMeta(posts)
	c.WriteIndex(posts)
	c.WritePosts(posts)
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

func (c *Client) CheckMeta(posts map[string]Post) bool {
	project := c.cfg.ReleaseProject
	username := c.cfg.ReleaseUserName
	branch := c.cfg.ReleaseBranch
	metafile := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/meta.json", username, project, branch)
	log.Println(metafile)
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

func (c *Client) WritePosts(posts map[string]Post) error {
	for _, post := range posts {
		log.Println(post)
		content, err := c.FetchContent(post.GUID)
		if err != nil {
			log.Println(err)
			continue
		}
		content, err = utils.Render(post.Title, content)
		if err != nil {
			log.Println(err)
			continue
		}
		conentWithImages, err := c.FilterImages(post.GUID, content)
		if err != nil {
			log.Println(err)
			continue
		}
		err = writeContent(c.cfg.ReleaseDir, post.Title, "html", conentWithImages)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}
func (c *Client) FilterImages(guid, content string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", err
	}
	//var images []string
	doc.Find("en-media").Each(func(i int, s *goquery.Selection) {
		hash, found := s.Attr("hash")
		if !found {
			return
		}
		typ, found := s.Attr("type")
		if !found {
			return
		}
		if strings.Contains(typ, "image") {
			binary, err := c.FetchBinary(guid, hash)
			log.Println("fetch binary image", len(binary), err)
			if err != nil {
				return
			}
			encoded := base64.StdEncoding.EncodeToString(binary)
			tpl := `<img src="data:%s;base64,%s"/>`
			image := fmt.Sprintf(tpl, typ, encoded)
			imageNode, err := html.Parse(strings.NewReader(image))
			if err != nil {
				return
			}
			s.Parent().AddNodes(imageNode)
			//images = append(images, image)
		}
	})
	return doc.Html()
}

func (c *Client) FetchBinary(guid, hash string) ([]byte, error) {
	store, err := c.client.GetNoteStore(c.token)
	if err != nil {
		return nil, err
	}
	noteguid := types.GUID(guid)
	res, err := store.GetResourceByHash(c.token, noteguid, []byte(hash), true, false, false)
	if err != nil {
		return nil, err
	}
	data := res.GetData()
	if data == nil {
		return nil, err
	}
	return data.Body, nil
}
func (c *Client) WriteMeta(posts map[string]Post) error {
	buf, _ := json.Marshal(posts)
	writeContent(c.cfg.ReleaseDir, "meta", "json", string(buf))
	return nil
}

func (c *Client) WriteIndex(posts map[string]Post) error {
	index := generateIndex(posts)
	writeContent(c.cfg.ReleaseDir, "index", "html", index)
	return nil
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
