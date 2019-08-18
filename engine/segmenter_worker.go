package engine

import (
	"fmt"
	"github.com/yanyiwu/gojieba"
	"octopus/types"
)

type SegmenterRequest struct {
	DocId       uint32
	Hash        uint32
	Data        types.DocumentIndexData
	ForceUpdate bool
}

func (engine *Engine) SegmenterWorker() {
	for {
		request := <-engine.segmenterChannel
		//if request.DocId == 0 {
		//	if request.ForceUpdate {
		//		var i uint32
		//		for i = 0; i < engine.initOptions.NumShards; i++ {
		//			engine.indexerAddDocChannels[i] <- IndexerAddDocumentRequest{forceUpdate: true}
		//		}
		//	}
		//	continue
		//}

		shard := engine.getShard(request.Hash)
		tokensMap := make(map[string]float32)
		numTokens := 0
		// 当文档正文不为空时, 从内容分词中得到关键词
		if request.Data.Content != "" {
			segments := gojieba.NewJieba().ExtractWithWeight(request.Data.Content, 1000)
			Normal := segments[0].Weight
			for _, segment := range segments {
				token := segment.Word
				tokensMap[token] = float32(segment.Weight / Normal)
			}
			numTokens = len(segments)
		} else {
			fmt.Println("content should not be empty!")
		}
		indexerRequest := IndexerAddDocumentRequest{
			document: &types.DocumentIndex{
				DocId:       request.DocId,
				TokenLength: float32(numTokens),
				Keywords:    make([]types.Keyword, len(tokensMap)),
			},
			forceUpdate: request.ForceUpdate,
		}
		iTokens := 0
		for k, v := range tokensMap {
			indexerRequest.document.Keywords[iTokens] = types.Keyword{
				Word:   k,
				Weight: v}
			iTokens++
		}

		engine.indexerAddDocChannels[shard] <- indexerRequest

		//if request.ForceUpdate {
		//	var i uint32
		//	for i = 0; i < engine.initOptions.NumShards; i++ {
		//		if i == shard {
		//			continue
		//		}
		//		engine.indexerAddDocChannels[i] <- IndexerAddDocumentRequest{forceUpdate: true}
		//	}
		//}
		//rankerRequest := rankerAddDocRequest{
		//	docId: request.DocId, fields: request.Data.Fields}
		//engine.rankerAddDocChannels[shard] <- rankerRequest
	}
}
