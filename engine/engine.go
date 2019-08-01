package engine

import (
	"fmt"
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
	// 初始化初始参数
	if engine.initialized {
		log.Fatal("请勿重复初始化引擎")
	}
	engine.initOptions = options
	engine.initialized = true

	// 将线程数设置为CPU数
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 初始化分词器通道
	engine.segmenterChannel = make(
		chan SegmenterRequest, options.NumSegmenterThreads)

	// 初始化索引器通道
	engine.indexerAddDocChannels = make(
		[]chan IndexerAddDocumentRequest, options.NumShards)
	// 初始化索引器
	for shard := 0; shard < options.NumShards; shard++ {
		engine.indexers = append(engine.indexers, core.Indexer{})
		engine.indexers[shard].Init(*options.IndexerInitOptions)
	}

	// 启动分词器
	for iThread := 0; iThread < options.NumSegmenterThreads; iThread++ {
		go engine.SegmenterWorker()
	}
}

// 将文档加入索引
//
// 输入参数：
//  docId	      标识文档编号，必须唯一，docId == 0 表示非法文档（用于强制刷新索引），[1, +oo) 表示合法文档
//  data	      见DocumentIndexData注释
//  forceUpdate 是否强制刷新 cache，如果设为 true，则尽快添加到索引，否则等待 cache 满之后一次全量添加
//
// 注意：
//      1. 这个函数是线程安全的，请尽可能并发调用以提高索引速度
//      2. 这个函数调用是非同步的，也就是说在函数返回时有可能文档还没有加入索引中，因此
//         如果立刻调用Search可能无法查询到这个文档。强制刷新索引请调用FlushIndex函数。
func (engine *Engine) IndexDocument(docId uint32, data types.DocumentIndexData, forceUpdate bool) {
	engine.internalIndexDocument(docId, data, forceUpdate)
}

func (engine *Engine) internalIndexDocument(docId uint32, data types.DocumentIndexData, forceUpdate bool) {
	if !engine.initialized {
		log.Fatal("必须先初始化引擎")
	}
	hash := murmur.Murmur3([]byte(fmt.Sprint("%d %s", docId, data.Content)))
	engine.segmenterChannel <- SegmenterRequest{
		DocId: docId, Hash: hash, Data: data, ForceUpdate: forceUpdate}
}

// 从文本hash得到要分配到的shard
func (engine *Engine) getShard(hash uint32) int {
	return int(hash - hash/uint32(engine.initOptions.NumShards)*uint32(engine.initOptions.NumShards))
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
