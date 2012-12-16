// Copyright 2012 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type AliasesService struct {
	client  *Client
	indices []string
	pretty  bool
	debug   bool
}

func NewAliasesService(client *Client) *AliasesService {
	builder := &AliasesService{
		client: client,
		indices: make([]string, 0),
	}
	return builder
}

func (s *AliasesService) Pretty(pretty bool) *AliasesService {
	s.pretty = pretty
	return s
}

func (s *AliasesService) Debug(debug bool) *AliasesService {
	s.debug = debug
	return s
}

func (s *AliasesService) Index(indexName string) *AliasesService {
	s.indices = append(s.indices, indexName)
	return s
}

func (s *AliasesService) Indices(indexNames ...string) *AliasesService {
	s.indices = append(s.indices, indexNames...)
	return s
}

func (s *AliasesService) Do() (*AliasesResult, error) {
	// Build url
	urls := "/"

	// Indices part
	indexPart := make([]string, 0)
	for _, index := range s.indices {
		indexPart = append(indexPart, cleanPathString(index))
	}
	urls += strings.Join(indexPart, ",")

	// TODO Types part

	// Search
	urls += "/_aliases"

	// Set up a new request
	req, err := s.client.NewRequest("GET", urls)
	if err != nil {
		return nil, err
	}

	// Parameters
	params := make(url.Values)
	if s.pretty {
		params.Set("pretty", fmt.Sprintf("%v", s.pretty))
	}
	urls += "?" + params.Encode()

	if s.debug {
		out, _ := httputil.DumpRequestOut((*http.Request)(req), true)
		fmt.Printf("%s\n", string(out))
	}

	// Get response
	res, err := s.client.c.Do((*http.Request)(req))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if s.debug {
		out, _ := httputil.DumpResponse(res, true)
		fmt.Printf("%s\n", string(out))
	}

	// {
	//   "indexName" : {
	//     "aliases" : {
	//       "alias1" : { },
	//       "alias2" : { }
	//     }
	//   },
	//   "indexName2" : {
	//     ...
	//   },
	// }
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	indexMap := make(map[string]interface{})
	if err := json.Unmarshal(bodyBytes, &indexMap); err != nil {
		return nil, err
	}

	// Each (indexName, _)
	ret := &AliasesResult{
		Indices: make(map[string]indexResult),
	}
	for indexName, indexData := range indexMap {
		indexOut, found := ret.Indices[indexName]
		if !found {
			indexOut = indexResult{Aliases: make([]aliasResult, 0)}
		}

		// { "aliases" : { ... } }
		indexDataMap, ok := indexData.(map[string]interface{})
		if ok {
			aliasesData, ok := indexDataMap["aliases"].(map[string]interface{})
			if ok {
				for aliasName, _ := range aliasesData {
					aliasRes := aliasResult{AliasName: aliasName}
					indexOut.Aliases = append(indexOut.Aliases, aliasRes)
				}
			}
		}

		ret.Indices[indexName] = indexOut
	}

	return ret, nil
}

// -- Result of an alias request.

type AliasesResult struct {
	Indices   map[string]indexResult
}

type indexResult struct {
	Aliases   []aliasResult
}

type aliasResult struct {
	AliasName string
}

func (ar AliasesResult) IndicesByAlias(aliasName string) []string {
	indices := make([]string, 0)

	for indexName, indexInfo := range ar.Indices {
		for _, aliasInfo := range indexInfo.Aliases {
			if aliasInfo.AliasName == aliasName {
				indices = append(indices, indexName)
			}
		} 
	}

	return indices
}

func (ir indexResult) HasAlias(aliasName string) bool {
	for _, alias := range ir.Aliases {
		if alias.AliasName == aliasName {
			return true
		}
	}
	return false
}
