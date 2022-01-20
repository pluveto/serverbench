package main

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"
)

type BasicProvider struct {
	Header map[string]string
	Body   []byte
}

func (p *BasicProvider) FromFile(headerPath string, bodyPath string) error {

	if len(headerPath) > 0 {
		sharedHeader_, err := ioutil.ReadFile(headerPath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(sharedHeader_, &p.Header)
		if err != nil {
			return err
		}
	}

	if len(bodyPath) > 0 {
		var err error
		p.Body, err = ioutil.ReadFile(bodyPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p BasicProvider) GetHeaders(ctx *RequestContext) map[string]string {
	return p.Header
}

func (p BasicProvider) GetBody(ctx *RequestContext) []byte {
	bodyStr := string(p.Body)
	replacedStr := strings.ReplaceAll(bodyStr, "${SEQ}", strconv.FormatInt(ctx.RequestSeq, 10))
	return []byte(replacedStr)
}
