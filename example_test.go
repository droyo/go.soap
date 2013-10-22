package soap

import (
	"encoding/xml"
	"fmt"
	"log"
)

var xmlData = []byte(`<Envelope>
  <Header>
    <sessionId href="#id0" />
  </Header>
  <Body>
    <multiRef id="id0">123456</multiRef>
  </Body>
</Envelope>`)

func ExampleUnmarshal() {
	var msg struct {
		XMLName xml.Name `xml:"Envelope"`
		Header  struct {
			Session string `xml:"sessionId"`
		}
	}

	if err := Unmarshal(xmlData, &msg); err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg.Header.Session)
	// Output:
	// 123456
}
