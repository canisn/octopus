package types

type RankOptions struct {
	// 默认情况下（ReverseOrder=false）按照分数从大到小排序，否则从小到大排序
	ReverseOrder bool

	// 从第几条结果开始输出
	OutputOffset int32

	// 最大输出的搜索结果数，为0时无限制
	MaxOutputs int32
}
