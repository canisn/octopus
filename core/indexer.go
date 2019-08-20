package core

import (
	"fmt"
	"log"
	"octopus/types"
	"sort"
	"sync"
)

type Indexer struct {
	// 从搜索键到文档列表的反向索引
	// 加了读写锁以保证读写安全
	tableLock struct {
		sync.RWMutex
		table map[string]*KeywordIndices
	}
	addCacheLock struct {
		sync.RWMutex
		addCachePointer uint32
		addCache        types.DocumentsIndex
	}

	initOptions IndexerInitOptions
	initialized bool

	// 这实际上是总文档数的一个近似
	numDocuments uint32

	// 所有被索引文本的总关键词数
	totalTokenLength float32

	// 每个文档的关键词长度
	docTokenLengths map[uint32]float32
}

// 反向索引表的一行，收集了一个搜索键出现的所有文档，按照DocId从小到大排序。
type KeywordIndices struct {
	docIds []uint32 // 全部类型都有
	weight []float32
}

// 初始化索引器
func (indexer *Indexer) Init(options IndexerInitOptions) {
	if indexer.initialized == true {
		log.Fatal("索引器不能初始化两次")
	}
	options.Init()
	indexer.initOptions = options
	indexer.initialized = true

	indexer.tableLock.table = make(map[string]*KeywordIndices)
	indexer.addCacheLock.addCache = make([]*types.DocumentIndex, indexer.initOptions.DocCacheSize)
	indexer.docTokenLengths = make(map[uint32]float32)
}

// 从KeywordIndices中得到第i个文档的DocId
func (indexer *Indexer) getDocId(ti *KeywordIndices, i uint32) uint32 {
	return ti.docIds[i]
}

// 得到KeywordIndices中文档总数
func (indexer *Indexer) getIndexLength(ti *KeywordIndices) uint32 {
	return uint32(len(ti.docIds))
}

// 向 ADDCACHE 中加入一个文档
func (indexer *Indexer) AddDocumentToCache(document *types.DocumentIndex, forceUpdate bool) {
	if indexer.initialized == false {
		log.Fatal("索引器尚未初始化")
	}

	indexer.addCacheLock.Lock()
	if document != nil {
		indexer.addCacheLock.addCache[indexer.addCacheLock.addCachePointer] = document
		indexer.addCacheLock.addCachePointer++
	}
	if indexer.addCacheLock.addCachePointer >= indexer.initOptions.DocCacheSize || forceUpdate {
		addCachedDocuments := indexer.addCacheLock.addCache[0:indexer.addCacheLock.addCachePointer]
		indexer.addCacheLock.addCachePointer = 0
		indexer.addCacheLock.Unlock()
		sort.Sort(addCachedDocuments)
		indexer.AddDocuments(&addCachedDocuments)
	} else {
		indexer.addCacheLock.Unlock()
	}
}

// 向反向索引表中加入 ADDCACHE 中所有文档
func (indexer *Indexer) AddDocuments(documents *types.DocumentsIndex) {
	if indexer.initialized == false {
		log.Fatal("索引器尚未初始化")
	}
	indexer.tableLock.Lock()
	defer indexer.tableLock.Unlock()
	indexPointers := make(map[string]uint32, len(indexer.tableLock.table))

	// DocId 递增顺序遍历插入文档保证索引移动次数最少
	for i, document := range *documents {
		if i < len(*documents)-1 && (*documents)[i].DocId == (*documents)[i+1].DocId {
			// 如果有重复文档加入，因为稳定排序，只加入最后一个
			fmt.Println("重复文档:", i)
			continue
		}

		// 更新文档关键词总长度
		if document.TokenLength != 0 {
			indexer.docTokenLengths[document.DocId] = float32(document.TokenLength)
			indexer.totalTokenLength += document.TokenLength
		}

		for index, keyword := range document.Keywords {
			indices, foundKeyword := indexer.tableLock.table[keyword.Word]
			if !foundKeyword {
				// 如果没找到该搜索键则加入
				//fmt.Printf("id:%d, keyword:%s, weight:%f\n", document.DocId, keyword.Word, document.Keywords[index].Weight)
				ti := KeywordIndices{}
				ti.docIds = []uint32{document.DocId}
				ti.weight = []float32{document.Keywords[index].Weight}
				indexer.tableLock.table[keyword.Word] = &ti
			} else {
				// 已有索引键
				position, _ := indexer.searchIndex(
					indices, indexPointers[keyword.Word], indexer.getIndexLength(indices)-1, document.DocId)
				indexPointers[keyword.Word] = position
				indices.docIds = append(indices.docIds, 0)
				copy(indices.docIds[position+1:], indices.docIds[position:])
				indices.docIds[position] = document.DocId

				indices.weight = append(indices.weight, 0)
				copy(indices.weight[position+1:], indices.weight[position:])
				indices.weight[position] = document.Keywords[index].Weight
			}
		}
		// 更新文章状态和总数
		indexer.numDocuments++
	}
	fmt.Println("indexer.numDocuments", indexer.numDocuments)
}

// 查找包含全部搜索键(AND操作)的文档
// 当docIds不为nil时仅从docIds指定的文档中查找
func (indexer *Indexer) Lookup(words []string) (docs PairList) {
	if indexer.initialized == false {
		log.Fatal("索引器尚未初始化")
	}

	if indexer.numDocuments == 0 {
		return
	}

	indexer.tableLock.RLock()
	defer indexer.tableLock.RUnlock()
	//table := make([]*KeywordIndices, len(keywords))
	table := make(map[uint32]float32)
	for _, word := range words {
		indices, found := indexer.tableLock.table[word]
		if !found {
			// 当反向索引表中无此搜索键时直接返回
			return
		} else {
			// 否则加入反向表中
			for index, docId := range indices.docIds {
				value, ok := table[docId]
				if ok {
					table[docId] = value + indices.weight[index]
				} else {
					table[docId] = indices.weight[index]
				}
			}

		}
	}
	docs = sortMapByValue(table)
	return
}

type Pair struct {
	Key   uint32
	Value float32
}
type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }
func sortMapByValue(m map[uint32]float32) PairList {
	p := make(PairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = Pair{k, v}
		i += 1
	}
	sort.Sort(p)
	return p
}

// 二分法查找indices中某文档的索引项
// 第一个返回参数为找到的位置或需要插入的位置
// 第二个返回参数标明是否找到
func (indexer *Indexer) searchIndex(
	indices *KeywordIndices, start uint32, end uint32, docId uint32) (uint32, bool) {
	// 特殊情况
	if indexer.getIndexLength(indices) == start {
		return start, false
	}
	if docId < indexer.getDocId(indices, start) {
		return start, false
	} else if docId == indexer.getDocId(indices, start) {
		return start, true
	}
	if docId > indexer.getDocId(indices, end) {
		return end + 1, false
	} else if docId == indexer.getDocId(indices, end) {
		return end, true
	}

	// 二分
	var middle uint32
	for end-start > 1 {
		middle = (start + end) / 2
		if docId == indexer.getDocId(indices, middle) {
			return middle, true
		} else if docId > indexer.getDocId(indices, middle) {
			start = middle
		} else {
			end = middle
		}
	}
	return end, false
}
