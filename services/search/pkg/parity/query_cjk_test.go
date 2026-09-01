package parity

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func cjkGroup() queryGroup {
	return queryGroup{
		name: "cjk",
		fixtures: []search.Resource{
			fixtureDoc("报告.txt", withContent("这是一个关于年度销售的中文文档")),
			fixtureDoc("说明书.pdf", withContent("产品说明与安装步骤"), withMime("application/pdf")),
			fixtureDoc("手册.txt", withContent("学生手册")),
			fixtureFolder("图片"),
			fixtureDoc("english.txt", withContent("annual sales report")),
		},
		cases: []queryCase{
			{id: 1, query: `Content:中文`, want: []string{"报告.txt"}},
			{id: 2, query: `Content:中文文档`, want: []string{"报告.txt"}},
			{id: 3, query: `Content:"中文文档"`, want: []string{"报告.txt"}},
			{id: 4, query: `Content:销售`, want: []string{"报告.txt"}},
			{id: 5, query: `Content:说明`, want: []string{"说明书.pdf"}},
			{id: 6, query: `name:报告`, want: []string{"报告.txt"}},
			{id: 7, query: `name:"*报告*"`, want: []string{"报告.txt"}},
			{id: 8, query: `name:"报告.txt"`, want: []string{"报告.txt"}},
			{id: 9, query: `报告`, want: []string{"报告.txt"}},
			{id: 10, query: `说明书`, want: []string{"说明书.pdf"}},
			{id: 11, query: `name:图片`, want: []string{"图片"}},
			{id: 12, query: `name:"*图片*"`, want: []string{"图片"}},
			{id: 13, query: `图片`, want: []string{"图片"}},
			{id: 14, query: `Content:学生`, want: []string{"手册.txt"}},
			{id: 15, query: `Content:手册`, want: []string{"手册.txt"}},
			{id: 16, query: `Content:生手`, want: []string{"手册.txt"}},
			{id: 17, query: `name:"报*"`, want: []string{"报告.txt"}},
			{id: 18, query: `name:"*告*"`, want: []string{"报告.txt"}},
			{id: 19, query: `name:"*册*"`, want: []string{"手册.txt"}},
			{id: 20, query: `name:"图?"`, want: []string{"图片"}},
			{id: 21, query: `name:"报?.txt"`, want: []string{"报告.txt"}},
			{id: 22, query: `name:"*报?*"`, want: []string{"报告.txt"}},
			{id: 23, query: `Content:中文*`},
			{id: 24, query: `Content:*销售*`},
			{id: 25, query: `Content:*文档`},
			{id: 26, query: `Content:文*`, want: []string{"报告.txt"}},
			{id: 27, query: `Content:*文*`, want: []string{"报告.txt"}},
			{id: 28, query: `Content:*文`, want: []string{"报告.txt"}},
			{id: 29, query: `Content:销*`, want: []string{"报告.txt"}},
			{id: 30, query: `Content:说*`, want: []string{"说明书.pdf"}},
			{id: 31, query: `Content:*明`, want: []string{"说明书.pdf"}},
		},
	}
}
