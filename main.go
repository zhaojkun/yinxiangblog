package main

import (
	"encoding/json"
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

type Post struct {
	GUID    string `json:"guid"`
	Title   string `json:"title"`
	Update  int64  `json:"update"`
	Content string `json:"-"`
}

func main() {
	dir := "public"
	token := os.Getenv("TOKEN")
	guid := os.Getenv("GUID")
	c := newClient(token)
	posts := c.GetPostList(guid)
	if changed := checkMeta(posts); !changed {
		return
	}
	writeContent("", "changed", "data", "true")
	buf, _ := json.Marshal(posts)
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

func checkMeta(posts map[string]Post) bool {
	resp, err := http.Get("https://raw.githubusercontent.com/zhaojkun/yinxiangblog/gh-pages/meta.json")
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
	token  string
	client *client.EvernoteClient
}

func newClient(token string) *Client {
	c := client.NewClient("", "", client.YINXIANG)
	cc := &Client{
		token:  token,
		client: c,
	}
	return cc
}

func (c *Client) GetPostList(guid string) map[string]Post {
	store, err := c.client.GetNoteStore(c.token)
	if err != nil {
		log.Fatal(err)
	}
	bloguuid := types.GUID(guid)
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
