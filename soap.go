// Package soap provides types and methods for decoding a subset of
// SOAP 1.1. The soap package closely mirrors the standard encoding/xml
// package. Unmarshaling rules are identical to that of encoding/xml,
// with the exception that document-local links are dereferenced.
package soap

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
)

const (
	NsXSI     = "http://www.w3.org/2001/XMLSchema-instance"
	NsXSD     = "http://www.w3.org/2001/XMLSchema"
	NsSoapEnv = "http://schemas.xmlsoap.org/soap/envelope/"
	Encoding  = "http://schemas.xmlsoap.org/soap/encoding/"
)

// A Fault describes a standard SOAP 1.1 Fault message.
type Fault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
	Actor   string   `xml:"faultactor"`
	Detail  []byte   `xml:"faultDetail"`
}

func (f *Fault) Error() string {
	if f == nil {
		return ""
	}
	return f.String
}

// NewRequest creates an http Request for use as a SOAP RPC
// call. The necessary SOAP headers are set.
func NewRequest(url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("SOAPAction", "")
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("charset", "utf-8")
	
	return req, nil
}

// Parse decodes an http response into a Go value. If the http
// response contains a SOAP Fault, an error is returned.
func Parse(resp *http.Response, v interface{}) error {
	var buf bytes.Buffer
	var msg struct {
		XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
		Body    struct {
			Fault *Fault
		} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	}
	
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return err
	}
	if err := xml.Unmarshal(buf.Bytes(), &msg); err != nil {
		return err
	}
	if msg.Body.Fault != nil {
		return msg.Body.Fault
	}
	return Unmarshal(buf.Bytes(), v)
}

// Unmarshal decodes XML data into a Go value. Unmarshal behaves identically
// to xml.Unmarshal, with the addition that document links are dereferenced.
func Unmarshal(data []byte, v interface{}) error {
	out, err := Flatten(data)
	if err != nil {
		return err
	}
	return xml.Unmarshal(out, v)
}

// Flatten reads XML data from a byte slice and returns a new XML
// document where all references have been replaced with copies of
// the referenced data.
func Flatten(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	mref, err := buildMRef(data)

	if err != nil {
		return nil, err
	}
	if elem, err := elements(data); err != nil {
		return nil, err
	} else {
		for _, el := range elem {
			data, err := flattenXML(el, mref)
			if err != nil {
				return nil, err
			}
			if _, err := buf.Write(data); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

//BUG(droyo) documents containing reference loops will probably kill
// the program. This is a security vulnerability and should be addressed
// before being put into production.
func flattenXML(root element, mref map[string]element) ([]byte, error) {
	var buf bytes.Buffer

	// heuristic for Apache axis 2 services
	if root.Name.Local == "multiRef" {
		return []byte(""), nil
	}

	if href, ok := findHref(root.Attr); ok {
		if el, ok := mref[href]; ok {
			root.Data = el.Data
		}
	}
	children := root.Children()
	if len(children) > 0 {
		var accum bytes.Buffer
		for _, el := range children {
			if data, err := flattenXML(el, mref); err != nil {
				return nil, err
			} else if _, err := accum.Write(data); err != nil {
				return nil, err
			}
		}
		root.Data = accum.Bytes()
	}
	if err := root.marshal(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
