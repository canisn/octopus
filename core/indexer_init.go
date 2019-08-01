package core

// 这些常数定义了反向索引表存储的数据类型
const (
	// 默认插入索引表文档 CACHE SIZE
	defaultDocCacheSize = 300000
)

// 初始化索引器选项
type IndexerInitOptions struct {
	// 索引表的类型，见上面的常数
	IndexType uint32

	// 待插入索引表文档 CACHE SIZE
	DocCacheSize int
}

func (options *IndexerInitOptions) Init() {
	if options.DocCacheSize == 0 {
		options.DocCacheSize = defaultDocCacheSize
	}
}
