package engine

import (
	"octopus/core"
)

var (
	// EngineInitOptions的默认值
	//NumCPU = runtime.NumCPU()
	numThread                               = 1
	defaultNumSegmenterThreads              = numThread
	defaultNumShards                 uint32 = 1
	defaultIndexerBufferLength              = numThread
	defaultNumIndexerThreadsPerShard        = numThread
	defaultRankerBufferLength               = numThread
	defaultNumRankerThreadsPerShard         = numThread
	defaultPersistentStorageShards          = 1
	defaultIndexerInitOptions               = core.IndexerInitOptions{}
)

type EngineInitOptions struct {
	// 分词器线程数
	NumSegmenterThreads int

	// 索引器和排序器的shard数目
	NumShards uint32

	// 索引器的信道缓冲长度
	IndexerBufferLength int

	// 索引器每个shard分配的线程数
	NumIndexerThreadsPerShard int

	// 排序器的信道缓冲长度
	RankerBufferLength int

	// 排序器每个shard分配的线程数
	NumRankerThreadsPerShard int

	// 索引器初始化选项
	IndexerInitOptions *core.IndexerInitOptions

	// 是否使用持久数据库，以及数据库文件保存的目录和裂分数目
	UsePersistentStorage    bool
	PersistentStorageFolder string
	PersistentStorageShards int
}

// 初始化EngineInitOptions，当用户未设定某个选项的值时用默认值取代
func (options *EngineInitOptions) Init() {

	if options.NumSegmenterThreads == 0 {
		options.NumSegmenterThreads = defaultNumSegmenterThreads
	}

	if options.NumShards == 0 {
		options.NumShards = defaultNumShards
	}

	if options.IndexerInitOptions == nil {
		options.IndexerInitOptions = &defaultIndexerInitOptions
	}

	if options.IndexerBufferLength == 0 {
		options.IndexerBufferLength = defaultIndexerBufferLength
	}

	if options.NumIndexerThreadsPerShard == 0 {
		options.NumIndexerThreadsPerShard = defaultNumIndexerThreadsPerShard
	}

	if options.RankerBufferLength == 0 {
		options.RankerBufferLength = defaultRankerBufferLength
	}

	if options.NumRankerThreadsPerShard == 0 {
		options.NumRankerThreadsPerShard = defaultNumRankerThreadsPerShard
	}

	if options.PersistentStorageShards == 0 {
		options.PersistentStorageShards = defaultPersistentStorageShards
	}
}
