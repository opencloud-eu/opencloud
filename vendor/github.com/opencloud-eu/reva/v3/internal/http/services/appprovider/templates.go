package appprovider

import (
	"strings"

	appregistry "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
)

type TemplateList struct {
	Templates map[string][]Template `json:"templates"`
}

type Template struct {
	Extension       string `json:"extension"`
	MimeType        string `json:"mime_type"`
	TargetExtension string `json:"target_extension"`
}

var tl = TemplateList{
	Templates: map[string][]Template{
		"collabora": {
			{
				MimeType:        "application/vnd.oasis.opendocument.spreadsheet-template",
				TargetExtension: "ods",
			},
			{
				MimeType:        "application/vnd.oasis.opendocument.text-template",
				TargetExtension: "odt",
			},
			{
				MimeType:        "application/vnd.oasis.opendocument.presentation-template",
				TargetExtension: "odp",
			},
		},
		"onlyoffice": {
			{
				MimeType:        "application/vnd.ms-word.template.macroenabled.12",
				TargetExtension: "docx",
			},
			{
				MimeType:        "application/vnd.oasis.opendocument.text-template",
				TargetExtension: "docx",
			},
			{
				MimeType:        "application/vnd.openxmlformats-officedocument.wordprocessingml.template",
				TargetExtension: "docx",
			},
			{
				MimeType:        "application/vnd.oasis.opendocument.spreadsheet-template",
				TargetExtension: "xlsx",
			},
			{
				MimeType:        "application/vnd.ms-excel.template.macroenabled.12",
				TargetExtension: "xlsx",
			},
			{
				MimeType:        "application/vnd.openxmlformats-officedocument.spreadsheetml.template",
				TargetExtension: "xlsx",
			},
			{
				MimeType:        "application/vnd.oasis.opendocument.presentation-template",
				TargetExtension: "pptx",
			},
			{
				MimeType:        "application/vnd.ms-powerpoint.template.macroenabled.12",
				TargetExtension: "pptx",
			},
			{
				MimeType:        "application/vnd.openxmlformats-officedocument.presentationml.template",
				TargetExtension: "pptx",
			},
		},
	},
}

func addTemplateInfo(mt *appregistry.MimeTypeInfo, apps []*ProviderInfo) {
	for _, app := range apps {
		if tls, ok := tl.Templates[strings.ToLower(app.ProductName)]; ok {
			for _, tmpl := range tls {
				if tmpl.Extension != "" && tmpl.Extension == mt.Ext {
					app.TargetExt = tmpl.TargetExtension
					continue
				}
				if tmpl.MimeType == mt.MimeType {
					app.TargetExt = tmpl.TargetExtension
				}
			}
		}
	}
}
