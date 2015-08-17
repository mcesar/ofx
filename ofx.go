package ofx

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Document type represents a OFX Document
type Document struct {
	Transactions []*Transaction `xml:"BANKMSGSRSV1>STMTTRNRS>STMTRS>BANKTRANLIST>STMTTRN"`
}

// Transaction type represents a transaction in a OFX Document
type Transaction struct {
	TxType      string  `xml:"TRNTYPE"`
	Date        date    `xml:"DTPOSTED"`
	Amount      float64 `xml:"TRNAMT"`
	FitID       string  `xml:"FITID"`
	Payee       string  `xml:"PAYEE NAME"`
	Memo        string  `xml:"MEMO"`
	Sic         string  `xml:"SIC"`
	CheckNumber string  `xml:"CHECKNUM"`
}

type date string

func (d date) String() string {
	s := string(d)
	return fmt.Sprintf("%v-%v-%v", s[0:4], s[4:6], s[6:8])
}

// Parse function returns a OFX Document representation given a Reader
func Parse(reader io.Reader) (*Document, error) {
	pr, pw := io.Pipe()
	defer pr.Close()
	go func() {
		brd := bufio.NewReader(reader)
		buf := make([]byte, 1)
		for {
			_, err := brd.Read(buf)
			if err == io.EOF {
				if closer, ok := reader.(io.Closer); ok {
					closer.Close()
				}
				pw.Close()
				return
			} else if err != nil {
				panic(err)
			}
			if buf[0] == '\r' {
				if buf2, err := brd.Peek(1); err != io.EOF && err != nil {
					panic(err)
				} else if buf2[0] == '\n' {
					continue
				} else {
					pw.Write([]byte{'\n'})
				}
			}
			_, err = pw.Write(buf)
			if err != nil {
				panic(err)
			}
		}
	}()
	sc := bufio.NewScanner(pr)
	var buf bytes.Buffer
	xmlStarted := false
	var transformer transform.Transformer
	tags := map[string]int{}
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 {
			xmlStarted = true
		}
		if xmlStarted {
			var l []byte
			if transformer != nil {
				rInUTF8 := transform.NewReader(strings.NewReader(line+"\n"), transformer)
				var err error
				l, err = ioutil.ReadAll(rInUTF8)
				if err != nil {
					return nil, err
				}
			} else {
				l = []byte(line + "\n")
			}
			if _, err := buf.Write(l); err != nil {
				return nil, err
			}
			tagEnding := strings.Index(string(l), "</")
			if tagEnding != -1 {
				tag := string(l)[tagEnding+2 : strings.Index(string(l), ">")]
				for i := len(tags) - 1; i >= 0; i-- {
					if _, ok := tags[tag]; ok {
						delete(tags, tag)
						break
					}
				}
			} else {
				tagBeginning := strings.Index(string(l), "<")
				if tagBeginning != -1 {
					tags[string(l)[tagBeginning+1:strings.Index(string(l), ">")]] = 0
				}
			}
		} else {
			if strings.TrimSpace(line) == "CHARSET:1252" {
				transformer = charmap.Windows1252.NewDecoder()
			}
		}
	}
	if sc.Err() != io.EOF && sc.Err() != nil {
		return nil, sc.Err()
	}
	var buf2 bytes.Buffer
	sc2 := bufio.NewScanner(&buf)
	for sc2.Scan() {
		line := sc2.Text()
		if _, err := buf2.Write([]byte(line)); err != nil {
			return nil, err
		}
		tagBeginning := strings.Index(line, "<")
		if tagBeginning != -1 {
			tag := line[tagBeginning+1 : strings.Index(line, ">")]
			if _, ok := tags[tag]; ok {
				if _, err := buf2.Write([]byte("</" + tag + ">")); err != nil {
					return nil, err
				}
			}
		}
	}
	if sc2.Err() != io.EOF && sc2.Err() != nil {
		return nil, sc2.Err()
	}
	doc := &Document{}
	dec := xml.NewDecoder(&buf2)
	if err := dec.Decode(doc); err != nil {
		return nil, err
	}
	return doc, nil
}
