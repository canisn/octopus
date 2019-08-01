package engine

import (
	"octopus/core"
	"octopus/types"
)

type EngineInitOptions struct {
	// 分词器线程数
	NumSegmenterThreads int

	// 索引器和排序器的shard数目
	NumShards int

	// 索引器的信道缓冲长度
	IndexerBufferLength uint32

	// 索引器每个shard分配的线程数
	NumIndexerThreadsPerShard uint32

	// 排序器的信道缓冲长度
	RankerBufferLength uint32

	// 排序器每个shard分配的线程数
	NumRankerThreadsPerShard uint32

	// 索引器初始化选项
	IndexerInitOptions *core.IndexerInitOptions

	// 默认的搜索选项
	DefaultRankOptions *types.RankOptions

	// 是否使用持久数据库，以及数据库文件保存的目录和裂分数目
	UsePersistentStorage    bool
	PersistentStorageFolder string
	PersistentStorageShards uint32
}
