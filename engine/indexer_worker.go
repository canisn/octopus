package engine

import (
	"fmt"
	"octopus/types"
	"sync/atomic"
)

type IndexerAddDocumentRequest struct {
	document    *types.DocumentIndex
	forceUpdate bool
}

func (engine *Engine) indexerAddDocumentWorker(shard uint32) {
	for {
		request := <-engine.indexerAddDocChannels[shard]
		fmt.Println("data to indexerAddDocChannels", request.document)
		engine.indexers[shard].AddDocumentToCache(request.document, request.forceUpdate)
		if request.document != nil {
			atomic.AddUint32(&engine.numTokenIndexAdded,
				uint32(len(request.document.Keywords)))
			atomic.AddUint32(&engine.numDocumentsIndexed, 1)
		}
		if request.forceUpdate {
			atomic.AddUint32(&engine.numDocumentsForceUpdated, 1)
		}
	}
}
