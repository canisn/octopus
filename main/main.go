package main

import (
	"octopus/engine"
	"time"
)

var (
	// searcher是线程安全的
	searcher = engine.Engine{}
)

func main() {
	// 初始化
	searcher.Init(engine.EngineInitOptions{})
	defer searcher.Close()
	//fmt.Println("searcher", searcher)
	searcher.IndexBulkDocumentFromMysql("127.0.0.1", "3306", "root", "root", "zhihudata", "zhihudata")
	//// 将文档加入索引，docId 从1开始
	//searcher.IndexDocument(1, types.DocumentIndexData{PostId: 12321, Title: "标题", Content: "文章正文当初正是因为" +
	//	"在微博上看到乔一关于你跟你男朋友是怎么确定恋爱关系的”的这个问题的回答，我与她结缘。发问题的微博是个大号，转发留言率都超高，" +
	//	"乔一这条回答被N个赞顶成了热门，我顺手点进她的微博主页，她微博粉丝只寥寥几十个，安安静静地记录着她与F君的生活小片段，看得正起" +
	//	"劲儿呢，三页翻到了尾，没了……然后，略感失落啊没看够啊……后来翻评论，发现很多人跟我一样，都是从那条热门微博评论摸都她微博来的，" +
	//	"纷纷在她微博下面留言——好有爱好萌好幸福啊以后多多发你们的有爱生活片段哟祝福",
	//	CreateTime: 152000000, UpdateTime: 152000001}, true)

	//等待索引刷新完毕
	//searcher.FlushIndex()
	time.Sleep(5 * time.Second)

	// 搜索输出格式见types.SearchResponse结构体
	//log.Print(searcher.Search(types.SearchRequest{Text: "百度中国"}))
}
