package soap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"text/template"
)

type element struct {
	xml.StartElement
	Data []byte `xml:",innerxml"`
}

func (el element) marshal(wr io.Writer) error {
	return xmlTmpl.ExecuteTemplate(wr, "Element", el)
}

// We want to re-use encoding/xml for parsing XML, but we want to be able
// to modify that XML (to de-reference links). So we need to be able to change
// our element structures back into XML text.
var xmlTmpl = template.Must(template.New("Marshal XML Elements").Parse(
`{{define "Name"}}{{if .Name.Space}}{{.Name.Space}}:{{end}}{{.Name.Local}}{{end}}
{{define "Attr"}}{{range .Attr}} {{template "Name" .}}="{{.Value}}"{{end}}{{end}}
{{define "StartTag"}}<{{template "Name" .}}{{template "Attr" .}}>{{end}}
{{define "EndTag"}}</{{template "Name" .}}>{{end}}
{{define "EmptyTag"}}<{{template "Name" .}}{{template "Attr" .}} />{{end}}
{{define "Element"}}{{if .Data}}{{template "StartTag" .}}{{printf "%s" .Data}}{{template "EndTag" .}}{{else}}{{template "EmptyTag" .}}{{end}}{{end}}`))

// Some routines for working with an XML document as a tree
func elements(data []byte) ([]element, error) {
	var (
		el   element
		elem []element
		tok  xml.Token
		err  error
		buf  bytes.Buffer
	)
	
	p := xml.NewDecoder(bytes.NewReader(data))
	for tok, err = p.RawToken(); err == nil; tok, err = p.RawToken() {
		if tok, ok := tok.(xml.StartElement); ok {
			el.StartElement = tok.Copy()
			if err := elementData(p, tok, &buf); err != nil {
				return nil, err
			}
			el.Data = make([]byte, buf.Len())
			copy(el.Data, buf.Bytes())
			elem = append(elem, el)
			buf.Reset()
		}
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	return elem, nil
}

// NOTE(droyo) we're walking the whole XML tree. We should consider
// collapsing buildMRef into this to do fewer passes on the document.
func elementData(p *xml.Decoder, start xml.StartElement, buf *bytes.Buffer) error {
	var tok xml.Token
	var err error

Loop:
	for tok, err = p.RawToken(); err == nil; tok, err = p.RawToken() {
		switch tok := tok.(type) {
		case xml.StartElement:
			if err := xmlTmpl.ExecuteTemplate(buf, "StartTag", tok); err != nil {
				return err
			}
			if err := elementData(p, tok, buf); err != nil {
				return err
			}
			if err := xmlTmpl.ExecuteTemplate(buf, "EndTag", tok); err != nil {
				return err
			}
		case xml.CharData:
			if err := xml.EscapeText(buf, tok); err != nil {
				return err
			}
		case xml.EndElement:
			if tok.Name == start.Name {
				break Loop
			} else {
				return errors.New("Unexpected end element " + tok.Name.Local)
			}
		default:
			continue
		}
	}
	return err
}

func (el element) Children() []element {
	if elem, err := elements(el.Data); err != nil {
		return nil
	} else {
		return elem
	}
}

func findAttr(list []xml.Attr, space, name string) *xml.Attr {
	for _, v := range list {
		if v.Name.Local == name && (space == "" || space == v.Name.Space) {
			return &v
		}
	}
	return nil
}

func findHref(list []xml.Attr) (string, bool) {
	attr := findAttr(list, "", "href")
	if attr != nil && len(attr.Value) > 1 && attr.Value[0] == '#' {
		return attr.Value[1:], true
	}
	return "", false
}

func findId(list []xml.Attr) (string, bool) {
	attr := findAttr(list, "", "id")
	if attr != nil {
		return attr.Value, true
	}
	return "", false
}

func buildMRef(data []byte) (map[string] element, error) {
	mref := make(map[string] element)
	
	elem, err := elements(data)
	if err != nil {
		return nil, err
	}
	
	for _, el := range elem {
		if err := walkMultiRef(el, mref); err != nil {
			return nil, err
		}
	}
	return mref, nil
}

func walkMultiRef(root element, mref map[string] element) error {
	children := root.Children()
	if len(children) > 0 {
		for _, el := range children {
			if err := walkMultiRef(el, mref); err != nil {
				return err
			}
		}
	}
	if id, ok := findId(root.Attr); ok {
		mref[id] = root
	}
	return nil
}
