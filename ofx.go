package ofx

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"strings"
)

type Document struct {
	Transactions []*Transaction `xml:"BANKMSGSRSV1>STMTTRNRS>STMTRS>BANKTRANLIST>STMTTRN"`
}

type Transaction struct {
	TxType      string  `xml:"TRNTYPE"`
	Date        Date    `xml:"DTPOSTED"`
	Amount      float64 `xml:"TRNAMT"`
	FitId       string  `xml:"FITID"`
	Payee       string  `xml:"PAYEE NAME"`
	Memo        string  `xml:"MEMO"`
	Sic         string  `xml:"SIC"`
	CheckNumber string  `xml:"CHECKNUM"`
}

type Date string

func (d Date) String() string {
	s := string(d)
	return fmt.Sprintf("%v-%v-%v", s[0:4], s[4:6], s[6:8])
}

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
	dec := xml.NewDecoder(&buf)
	xmlStarted := false
	var transformer transform.Transformer
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 {
			xmlStarted = true
		}
		if xmlStarted {
			var l []byte
			if transformer != nil {
				rInUTF8 := transform.NewReader(strings.NewReader(line), transformer)
				var err error
				l, err = ioutil.ReadAll(rInUTF8)
				if err != nil {
					return nil, err
				}
			} else {
				l = []byte(line)
			}
			if _, err := buf.Write(l); err != nil {
				return nil, err
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
	doc := &Document{}
	if err := dec.Decode(doc); err != nil {
		return nil, err
	}
	return doc, nil
}
