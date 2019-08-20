package main

import (
	"database/sql"
	"fmt"
	"octopus/engine"
	"octopus/types"
)

var (
	// searcher是线程安全的
	searcher = engine.Engine{}
)

func main() {
	// 初始化
	searcher.Init(engine.EngineInitOptions{
		//UsePersistentStorage:    true,
		//PersistentStorageFolder: "data",
	})
	defer searcher.Close()
	//searcher.IndexBulkDocumentFromMysql("127.0.0.1", "3306", "root", "root", "zhihudata", "zhihudata")
	// 将文档加入索引，docId 从1开始
	for a := 1; a < 550; a++ {
		searcher.IndexDocument(uint64(a), types.DocumentIndexData{PostId: 12321, Title: "标题", Content: "文章正文当初正是因为" +
			"在微博上看到乔一关于你跟你男朋友是怎么确定恋爱关系的”的这个问题的回答，我与她结缘。发问题的微博是个大号，转发留言率都超高，" +
			"乔一这条回答被N个赞顶成了热门，我顺手点进她的微博主页，她微博粉丝只寥寥几十个，安安静静地记录着她与F君的生活小片段，看得正起" +
			"劲儿呢，三页翻到了尾，没了……然后，略感失落啊没看够啊……后来翻评论，发现很多人跟我一样，都是从那条热门微博评论摸都她微博来的，" +
			"纷纷在她微博下面留言——好有爱好萌好幸福啊以后多多发你们的有爱生活片段哟祝福",
			CreateTime: 152000000, UpdateTime: 152000001}, false)
	}
	//等待索引刷新完毕
	searcher.FlushIndex()
	var text string
	for ; ; {
		fmt.Printf("请输入关键词: ")
		fmt.Scanln(&text) //Scanln 扫描来自标准输入的文本，将空格分隔的值依次存放到后续的参数内，直到碰到换行
		fmt.Println("查询结果为：")
		pairlist := searcher.Search(types.SearchRequest{Text: text})
		for _, v := range pairlist {
			fmt.Println("----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
			fmt.Println("帖子id", v.Key)
			fmt.Println("评分:", v.Value)
			//ReadMysql("127.0.0.1", "3306", v.Key)
			fmt.Println("----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
			fmt.Println()
		}
	}
}

//从mysql获取文档加入索引
func ReadMysql(mysql_ip string, mysql_port string, id uint32) {
	//打开数据库
	//fmt.Print(user+":"+password+"@tcp("+host+")/"+dbName+"?charset=utf8")
	db, errOpen := sql.Open("mysql", "root:root@tcp("+mysql_ip+":"+mysql_port+")/zhihudata?charset=utf8")
	if errOpen != nil {
		//TODO，这里只是打印了一下，并没有做异常处理
		fmt.Println("funReadSql Open is error", errOpen)
	}

	//读取t_knowledge_tree表中codeName和answer字段
	rows, err := db.Query("select id,pid,title,excerpt from zhihudata where id=? ", id)
	if err != nil {
		fmt.Println("error:", err)
	}
	for rows.Next() {
		var id uint32
		var pid uint32
		var title string
		var excerpt string
		err = rows.Scan(&id, &pid, &title, &excerpt)

		fmt.Printf("知乎连接:https://zhuanlan.zhihu.com/p/%d", pid)
		fmt.Println()
		fmt.Println("文章标题:", title)
		fmt.Println("文章摘要:", excerpt)
	}

	//start += 10000
	//fmt.Print(start)
	if err != nil {
		//TODO，这里只是打印了一下，并没有做异常处理
		fmt.Println("funReadSql SELECT t_knowledge_tree is error", err)
	}
	//关闭数据库
	db.Close()
}
