package engine

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/huichen/murmur"
	"log"
	"octopus/core"
	"octopus/storage"
	"octopus/types"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
)

const PersistentStorageFilePrefix = "zuiyou"

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

	dbs []storage.Storage
	//建立分词器通道
	segmenterChannel chan SegmenterRequest

	// 建立索引器使用的通信通道
	indexerAddDocChannels []chan IndexerAddDocumentRequest

	// 建立持久存储使用的通信通道
	persistentStorageIndexDocumentChannels []chan persistentStorageIndexDocumentRequest
	persistentStorageInitChannel           chan bool
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
	// 初始化持久化存储通道
	if engine.initOptions.UsePersistentStorage {
		engine.persistentStorageIndexDocumentChannels =
			make([]chan persistentStorageIndexDocumentRequest,
				engine.initOptions.PersistentStorageShards)
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			engine.persistentStorageIndexDocumentChannels[shard] = make(
				chan persistentStorageIndexDocumentRequest)
		}
		engine.persistentStorageInitChannel = make(
			chan bool, engine.initOptions.PersistentStorageShards)
	}
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

	// 启动持久化存储工作协程
	if engine.initOptions.UsePersistentStorage {
		fmt.Print("初始化")
		//创建文件夹
		err := os.MkdirAll(engine.initOptions.PersistentStorageFolder, 0700)
		if err != nil {
			log.Fatal("无法创建目录", engine.initOptions.PersistentStorageFolder)
		}

		// 打开或者创建数据库
		engine.dbs = make([]storage.Storage, engine.initOptions.PersistentStorageShards) //创建数组
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			dbPath := engine.initOptions.PersistentStorageFolder + "/" + PersistentStorageFilePrefix + "." + strconv.Itoa(shard)
			db, err := storage.OpenStorage(dbPath)
			if db == nil || err != nil {
				log.Fatal("无法打开数据库", dbPath, ": ", err)
			}
			engine.dbs[shard] = db
		}

		// 从数据库中恢复
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			go engine.persistentStorageInitWorker(shard)
		}

		// 等待恢复完成
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			<-engine.persistentStorageInitChannel
		}
		for {
			runtime.Gosched()
			if engine.numIndexingRequests == engine.numDocumentsIndexed {
				break
			}
		}

		// 关闭并重新打开数据库
		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			engine.dbs[shard].Close()
			dbPath := engine.initOptions.PersistentStorageFolder + "/" + PersistentStorageFilePrefix + "." + strconv.Itoa(shard)
			db, err := storage.OpenStorage(dbPath)
			if db == nil || err != nil {
				log.Fatal("无法打开数据库", dbPath, ": ", err)
			}
			engine.dbs[shard] = db
		}

		for shard := 0; shard < engine.initOptions.PersistentStorageShards; shard++ {
			go engine.persistentStorageIndexDocumentWorker(shard)
		}
	}
}

// 将文档加入索引
// 输入参数：
//  docId	      标识文档编号，必须唯一，docId == 0 表示非法文档（用于强制刷新索引），[1, +oo) 表示合法文档
//  data	      见DocumentIndexData注释
//  forceUpdate 是否强制刷新 cache，如果设为 true，则尽快添加到索引，否则等待 cache 满之后一次全量添加

func (engine *Engine) IndexDocument(docId uint64, data types.DocumentIndexData, forceUpdate bool) {
	engine.internalIndexDocument(docId, data, forceUpdate)

	hash := murmur.Murmur3([]byte(fmt.Sprint("%d", docId))) % uint32(engine.initOptions.PersistentStorageShards)
	if engine.initOptions.UsePersistentStorage && docId != 0 {
		engine.persistentStorageIndexDocumentChannels[hash] <- persistentStorageIndexDocumentRequest{docId: docId, data: data}
	}
}

func (engine *Engine) internalIndexDocument(
	docId uint64, data types.DocumentIndexData, forceUpdate bool) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}

	if docId != 0 {
		atomic.AddUint32(&engine.numIndexingRequests, 1)
	}
	if forceUpdate {
		atomic.AddUint32(&engine.numForceUpdatingRequests, 1)
	}
	hash := murmur.Murmur3([]byte(fmt.Sprint("%d%s", docId, data.Content)))
	engine.segmenterChannel <- SegmenterRequest{
		DocId: uint32(docId), Hash: hash, Data: data, ForceUpdate: forceUpdate}
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
		rows, err := db.Query("select id,pid,title,content,created,updated from zhihudata order by id limit ?,10 ", start)
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

func (engine *Engine) Print() {
	fmt.Print(engine.numDocumentsIndexed)
	fmt.Print(">.......")
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
