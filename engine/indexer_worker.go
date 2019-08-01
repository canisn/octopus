package engine

import "octopus/types"


type IndexerAddDocumentRequest struct {
	document    *types.DocumentIndex
	forceUpdate bool
}
