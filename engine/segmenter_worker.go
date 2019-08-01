package engine

import (
	"fmt"
	"octopus/types"
	"github.com/yanyiwu/gojieba"
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
		fmt.Println("SegmenterWorker read data from segmenterChannel", request.DocId, request.Data)
		if request.DocId == 0 {
			if request.ForceUpdate {
				for i := 0; i < engine.initOptions.NumShards; i++ {
					engine.indexerAddDocChannels[i] <- IndexerAddDocumentRequest{forceUpdate: true}
				}
			}
			continue
		}

		shard := engine.getShard(request.Hash)
		tokensMap := make(map[string][]float32)
		numTokens := 0
		// 当文档正文不为空时, 从内容分词中得到关键词
		if request.Data.Content != "" {
			segments := gojieba.NewJieba().ExtractWithWeight(request.Data.Content, 1000)
			for _, segment := range segments {
				token := segment.Word
				tokensMap[token] = append(tokensMap[token], float32(segment.Weight))
			}
			numTokens = len(segments)
		} else {
			continue
		}

		indexerRequest := IndexerAddDocumentRequest{
			document: &types.DocumentIndex{
				DocId:       request.DocId,
				TokenLength: float32(numTokens),
				Keywords:    make([]types.Keyword, len(tokensMap)),
			},
			forceUpdate: request.ForceUpdate,
		}

		engine.indexerAddDocChannels[shard] <- indexerRequest
		if request.ForceUpdate {
			for i := 0; i < engine.initOptions.NumShards; i++ {
				if i == shard {
					continue
				}
				engine.indexerAddDocChannels[i] <- IndexerAddDocumentRequest{forceUpdate: true}
			}
		}
		//rankerRequest := rankerAddDocRequest{
		//	docId: request.DocId, fields: request.Data.Fields}
		//engine.rankerAddDocChannels[shard] <- rankerRequest
	}
}
