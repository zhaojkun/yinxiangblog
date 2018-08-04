package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"

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
	for _, post := range posts {
		log.Println(post)
		content, err := c.FetchContent(post.GUID)
		if err != nil {
			log.Println(err)
			continue
		}
		err = writeContent(dir, post.Title, content)
		if err != nil {
			log.Println(err)
		}
	}
}

func writeContent(dir, title, content string) error {
	os.MkdirAll(dir, 0644)
	p := path.Join(dir, title+".html")
	return ioutil.WriteFile(p, []byte(content), 0644)
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
