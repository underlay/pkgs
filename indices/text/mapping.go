package text

import (
	"log"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/analysis/char/asciifolding"
	"github.com/blevesearch/bleve/analysis/char/regexp"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/token/ngram"
	"github.com/blevesearch/bleve/analysis/tokenizer/letter"
	"github.com/blevesearch/bleve/mapping"

	types "github.com/underlay/pkgs/types"
)

func getMapping() mapping.IndexMapping {
	mapping := bleve.NewIndexMapping()

	err := mapping.AddCustomCharFilter("alphanumeric", map[string]interface{}{
		"type":    regexp.Name,
		"regexp":  "[^A-Za-z0-9]*",
		"replace": "",
	})

	if err != nil {
		log.Fatalln(err)
	}

	err = mapping.AddCustomTokenFilter("title_ngrams", map[string]interface{}{
		"type": ngram.Name,
		"min":  3,
		"max":  12,
	})

	if err != nil {
		log.Fatalln(err)
	}

	err = mapping.AddCustomAnalyzer("title", map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": letter.Name,
		"char_filters": []interface{}{
			asciifolding.Name,
			"alphanumeric",
		},
		"token_filters": []interface{}{
			lowercase.Name,
			"title_ngrams",
		},
	})

	if err != nil {
		log.Fatalln(err)
	}

	idField := bleve.NewTextFieldMapping()
	idField.Name = "id"
	idField.Analyzer = keyword.Name

	titleField := bleve.NewTextFieldMapping()
	titleField.Name = "title"
	titleField.Analyzer = "title"

	descriptionField := bleve.NewTextFieldMapping()
	descriptionField.Name = "description"
	descriptionField.Analyzer = standard.Name

	packageMapping := bleve.NewDocumentMapping()
	packageMapping.Dynamic = false
	packageMapping.AddFieldMappingsAt("id", idField)
	packageMapping.AddFieldMappingsAt("title", titleField)
	packageMapping.AddFieldMappingsAt("description", descriptionField)
	mapping.AddDocumentMapping(types.LDPDirectContainer, packageMapping)

	assertionMapping := bleve.NewDocumentMapping()
	assertionMapping.Dynamic = false
	assertionMapping.AddFieldMappingsAt("id", idField)
	assertionMapping.AddFieldMappingsAt("title", titleField)
	mapping.AddDocumentMapping(types.LDPRDFSource, assertionMapping)

	fileMapping := bleve.NewDocumentMapping()
	fileMapping.Dynamic = false
	fileMapping.AddFieldMappingsAt("id", idField)
	fileMapping.AddFieldMappingsAt("title", titleField)
	mapping.AddDocumentMapping(types.LDPNonRDFSource, fileMapping)
	return mapping
}
