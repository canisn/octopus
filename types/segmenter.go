package types

type SegmenterRequest struct {
	docId       uint64
	hash        uint32
	data        DocumentIndexData
	forceUpdate bool
}