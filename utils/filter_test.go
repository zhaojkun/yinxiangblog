package utils

import (
	"fmt"
	"testing"
)

func TestFilter(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE en-note SYSTEM "http://xml.evernote.com/pub/enml2.dtd">

<en-note><div>here is a image</div><div><en-media hash="f5dc113a0ce391202b55d8c4fa580a0e" type="image/png"/></div><div><br/></div></en-note>
`
	res, err := Filter("test pic", content)
	fmt.Println(err)
	fmt.Println(res)
}
