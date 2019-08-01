package types

type DocumentIndexData struct {
	//文章识别符
	PostId uint32
	//标题
	Title string
	//文档全文（必须是UTF-8格式），用于生成待索引的关键词
	Content string
	//创建时间
	CreateTime uint32
	//更新时间
	UpdateTime uint32
}

type DocumentIndex struct {
	// 文本的DocId
	DocId uint32

	// 文本的关键词长
	TokenLength float32

	// 加入的索引键
	Keywords []Keyword
}

// 文档的一个关键词
type Keyword struct {
	//关键词的字符串
	Text string
	//权重
	Weight float32
}

func (docs DocumentsIndex) Len() int {
	return len(docs)
}
func (docs DocumentsIndex) Swap(i, j int) {
	docs[i], docs[j] = docs[j], docs[i]
}
func (docs DocumentsIndex) Less(i, j int) bool {
	return docs[i].DocId < docs[j].DocId
}
// 方便批量加入文档索引
type DocumentsIndex []*DocumentIndex