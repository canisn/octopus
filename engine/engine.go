package engine

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/huichen/murmur"
	"log"
	"octopus/core"
	"octopus/types"
	"runtime"
)

type Engine struct {
	// 计数器，用来统计有多少文档被索引等信息
	numDocumentsIndexed      uint32
	numDocumentsRemoved      uint32
	numDocumentsForceUpdated uint32
	numIndexingRequests      uint32
	numRemovingRequests      uint32
	numForceUpdatingRequests uint32
	numTokenIndexAdded       uint32
	numDocumentsStored       uint32
	// 记录初始化参数
	initOptions EngineInitOptions
	initialized bool

	// 索引器
	indexers []core.Indexer

	//建立分词器通道
	segmenterChannel chan SegmenterRequest

	// 建立索引器使用的通信通道
	indexerAddDocChannels []chan IndexerAddDocumentRequest
}

func (engine *Engine) Init(options EngineInitOptions) {
	// 将线程数设置为CPU数
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 初始化初始参数
	if engine.initialized {
		log.Fatal("请勿重复初始化引擎")
	}
	options.Init()
	fmt.Println(options)
	engine.initOptions = options
	engine.initialized = true

	// 初始化分词器通道
	engine.segmenterChannel = make(
		chan SegmenterRequest, options.NumSegmenterThreads)

	// 启动分词器
	for iThread := 0; iThread < options.NumSegmenterThreads; iThread++ {
		go engine.SegmenterWorker()
		fmt.Println("SegmenterWorker start")
	}

	// 初始化索引器
	var i uint32
	for i = 0; i < options.NumShards; i++ {
		engine.indexers = append(engine.indexers, core.Indexer{})
		engine.indexers[i].Init(*options.IndexerInitOptions)
	}
	// 初始化索引器通道
	engine.indexerAddDocChannels = make(
		[]chan IndexerAddDocumentRequest, options.NumShards)

	for i = 0; i < options.NumShards; i++ {
		engine.indexerAddDocChannels[i] = make(
			chan IndexerAddDocumentRequest,
			options.IndexerBufferLength)
	}

	// 启动索引器
	for i = 0; i < options.NumShards; i++ {
		go engine.indexerAddDocumentWorker(i)
		fmt.Println("indexerAddDocumentWorker start")
	}
	fmt.Println("engine ", engine.initOptions, engine.segmenterChannel, engine.indexerAddDocChannels[0])
}

// 将文档加入索引
// 输入参数：
//  docId	      标识文档编号，必须唯一，docId == 0 表示非法文档（用于强制刷新索引），[1, +oo) 表示合法文档
//  data	      见DocumentIndexData注释
//  forceUpdate 是否强制刷新 cache，如果设为 true，则尽快添加到索引，否则等待 cache 满之后一次全量添加

func (engine *Engine) IndexDocument(docId uint32, data types.DocumentIndexData, forceUpdate bool) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}
	hash := murmur.Murmur3([]byte(fmt.Sprint("%d %s", docId, data.Content)))
	engine.segmenterChannel <- SegmenterRequest{
		DocId: docId, Hash: hash, Data: data, ForceUpdate: forceUpdate}
}

//从mysql获取文档加入索引
func (engine *Engine) IndexBulkDocumentFromMysql(mysql_ip string, mysql_port string, mysql_user string, mysql_passwd string, mysql_qyDB string, table string) {
	//打开数据库
	//fmt.Print(user+":"+password+"@tcp("+host+")/"+dbName+"?charset=utf8")
	db, errOpen := sql.Open("mysql", mysql_user+":"+mysql_passwd+"@tcp("+mysql_ip+":"+mysql_port+")/"+mysql_qyDB+"?charset=utf8")
	if errOpen != nil {
		//TODO，这里只是打印了一下，并没有做异常处理
		fmt.Println("funReadSql Open is error", errOpen)
	}

	start := 0
	for {
		//读取t_knowledge_tree表中codeName和answer字段
		rows, err := db.Query("select id,pid,title,content,created,updated from zhihudata order by id limit ?,10000 ", start)
		if err != nil {
			fmt.Println("error:", err)
		}
		flag := false
		for rows.Next() {
			var id uint32
			var pid uint32
			var title string
			var content string
			var createtime uint32
			var updatetime uint32
			err = rows.Scan(&id, &pid, &title, &content, &createtime, &updatetime)
			data := types.DocumentIndexData{PostId: pid, Title: title, Content: content,
				CreateTime: createtime, UpdateTime: updatetime}
			hash := murmur.Murmur3([]byte(fmt.Sprint("%d %s", id, data.Content)))
			engine.segmenterChannel <- SegmenterRequest{
				DocId: id, Hash: hash, Data: data, ForceUpdate: false}
			flag = true
		}
		if !flag {
			break
		}
		start += 10000
		if err != nil {
			//TODO，这里只是打印了一下，并没有做异常处理
			fmt.Println("funReadSql SELECT t_knowledge_tree is error", err)
		}
	}
	//关闭数据库
	db.Close()

}

// 从文本hash得到要分配到的shard
func (engine *Engine) getShard(hash uint32) uint32 {
	return hash % engine.initOptions.NumShards
	//return int(hash - hash/uint32(engine.initOptions.NumShards)*uint32(engine.initOptions.NumShards))
}

// 阻塞等待直到所有索引添加完毕
func (engine *Engine) FlushIndex() {
	for {
		runtime.Gosched()
		if engine.numIndexingRequests == engine.numDocumentsIndexed &&
			engine.numRemovingRequests*uint32(engine.initOptions.NumShards) == engine.numDocumentsRemoved &&
			(!engine.initOptions.UsePersistentStorage || engine.numIndexingRequests == engine.numDocumentsStored) {
			// 保证 CHANNEL 中 REQUESTS 全部被执行完
			break
		}
	}
	// 强制更新，保证其为最后的请求
	engine.IndexDocument(0, types.DocumentIndexData{}, true)
	for {
		runtime.Gosched()
		if engine.numForceUpdatingRequests*uint32(engine.initOptions.NumShards) == engine.numDocumentsForceUpdated {
			return
		}
	}
}

// 关闭引擎
func (engine *Engine) Close() {
	engine.FlushIndex()
	//if engine.initOptions.UsePersistentStorage {
	//	for _, db := range engine.dbs {
	//		db.Close()
	//	}
	//}
}
