package ofx

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
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
	sc := bufio.NewScanner(reader)
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}
	var buf bytes.Buffer
	dec := xml.NewDecoder(&buf)
	xmlStarted := false
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 {
			xmlStarted = true
		}
		if xmlStarted {
			if _, err := buf.Write([]byte(line + "\n")); err != nil {
				return nil, err
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
