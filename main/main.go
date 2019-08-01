package main

import (
	"octopus/engine"
	"octopus/types"
)

var (
	// searcher是线程安全的
	searcher = engine.Engine{}
)

func main() {
	// 初始化
	searcher = engine.Engine{}
	searcher.Init(engine.EngineInitOptions{})
	defer searcher.Close()

	// 将文档加入索引，docId 从1开始
	searcher.IndexDocument(1, types.DocumentIndexData{PostId: 12321, Title: "标题", Content: "文章正文",
		CreateTime: 1520000000, UpdateTime: 1520000001}, false)

	// 等待索引刷新完毕
	searcher.FlushIndex()

	// 搜索输出格式见types.SearchResponse结构体
	//log.Print(searcher.Search(types.SearchRequest{Text: "百度中国"}))
}
