package types


type IndexerAddDocumentRequest struct {
	document    *DocumentIndex
	forceUpdate bool
}
