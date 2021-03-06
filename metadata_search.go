// Copyright 2020 Pexeso Inc. All rights reserved.

package pexae

// #include <pex/ae/sdk/c/metadata_search.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"sync"
)

// Holds all data necessary to perform a metadata search. A search can only be
// performed using a fingerprint, but additional parameters may be supported in
// the future.
type MetadataSearchRequest struct {
	// A fingerprint obtained by calling either NewFingerprintFromFile
	// or NewFingerprintFromBuffer. This field is required.
	Fingerprint *Fingerprint
}

// This object is returned from MetadataSearchFuture.Get upon successful
// completion.
type MetadataSearchResult struct {
	// An ID that uniquely identifies a particular search. Can be used
	// for diagnostics.
	LookupID uint64

	// An ID that uniquely identifies the UGC. It is used to provide UGC metadata back to Pex.
	UGCID uint64

	// A list of matches.
	Matches []*MetadataSearchMatch
}

// MetadataSearchMatch contains detailed information about the match,
// including information about the matched asset, and the matching
// segments.
type MetadataSearchMatch struct {
	// An ID that uniquely identifies a matching asset. This can be used to
	// retrieve detailed information about the asset using
	// AssetLibrary.GetAsset.
	AssetID uint64

	// A list of matching segments.
	Segments []*Segment
}

// This class encapsulates all operations necessary to perform a
// metadata search. Instead of instantiating the class directly,
// Client.MetadataSearch should be used.
type MetadataSearch struct {
	c *C.AE_MetadataSearch
}

// Start starts a metadata search. This operation does not block until
// the search is finished, it does however perform a network operation
// to initiate the search on the backend service.
func (x *MetadataSearch) Start(req *MetadataSearchRequest) (*MetadataSearchFuture, error) {
	cStatus := C.AE_Status_New()
	if cStatus == nil {
		panic("out of memory")
	}
	defer C.AE_Status_Delete(&cStatus)

	cRequest := C.AE_MetadataSearchRequest_New()
	if cRequest == nil {
		panic("out of memory")
	}
	defer C.AE_MetadataSearchRequest_Delete(&cRequest)

	cFuture := C.AE_MetadataSearchFuture_New()
	if cFuture == nil {
		panic("out of memory")
	}

	C.AE_MetadataSearchRequest_SetFingerprint(cRequest, req.Fingerprint.ft)

	C.AE_MetadataSearch_Start(x.c, cRequest, cFuture, cStatus)
	if err := statusToError(cStatus); err != nil {
		// Delete the resource here to prevent leaking.
		C.AE_MetadataSearchFuture_Delete(&cFuture)
		return nil, err
	}

	return &MetadataSearchFuture{
		LookupID: uint64(C.AE_MetadataSearchFuture_GetLookupID(cFuture)),
		c:        cFuture,
	}, nil
}

// MetadataSearchFuture object is returned by the MetadataSearch.Start
// function and is used to retrieve a search result.
type MetadataSearchFuture struct {
	LookupID uint64

	c *C.AE_MetadataSearchFuture
	m sync.Mutex
}

// Get blocks until the search result is ready and then returns it. It
// also releases all the allocated resources, so it will return an
// error when called multiple times.
func (x *MetadataSearchFuture) Get() (*MetadataSearchResult, error) {
	x.m.Lock()
	defer x.m.Unlock()

	if x.c == nil {
		return nil, errors.New("already called")
	}
	defer x.close()

	cStatus := C.AE_Status_New()
	if cStatus == nil {
		panic("out of memory")
	}
	defer C.AE_Status_Delete(&cStatus)

	cResult := C.AE_MetadataSearchResult_New()
	if cResult == nil {
		panic("out of memory")
	}
	defer C.AE_MetadataSearchResult_Delete(&cResult)

	C.AE_MetadataSearchFuture_Get(x.c, cResult, cStatus)
	if err := statusToError(cStatus); err != nil {
		return nil, err
	}
	return x.processResult(cResult), nil
}

func (x *MetadataSearchFuture) close() {
	C.AE_MetadataSearchFuture_Delete(&x.c)
	x.c = nil
}

func (x *MetadataSearchFuture) processResult(cResult *C.AE_MetadataSearchResult) *MetadataSearchResult {
	cMatch := C.AE_MetadataSearchMatch_New()
	if cMatch == nil {
		panic("out of memory")
	}
	defer C.AE_MetadataSearchMatch_Delete(&cMatch)

	var cMatchesPos C.size_t = 0
	var matches []*MetadataSearchMatch

	for C.AE_MetadataSearchResult_NextMatch(cResult, cMatch, &cMatchesPos) {
		var cQueryStart C.int64_t
		var cQueryEnd C.int64_t
		var cAssetStart C.int64_t
		var cAssetEnd C.int64_t
		var cSegmentsPos C.size_t = 0
		var segments []*Segment

		for C.AE_MetadataSearchMatch_NextSegment(cMatch, &cQueryStart, &cQueryEnd, &cAssetStart, &cAssetEnd, &cSegmentsPos) {
			segments = append(segments, &Segment{
				QueryStart: int64(cQueryStart),
				QueryEnd:   int64(cQueryEnd),
				AssetStart: int64(cAssetStart),
				AssetEnd:   int64(cAssetEnd),
			})
		}

		matches = append(matches, &MetadataSearchMatch{
			AssetID:  uint64(C.AE_MetadataSearchMatch_GetAssetID(cMatch)),
			Segments: segments,
		})
	}

	return &MetadataSearchResult{
		LookupID: uint64(C.AE_MetadataSearchResult_GetLookupID(cResult)),
		UGCID:    uint64(C.AE_MetadataSearchResult_GetUGCID(cResult)),
		Matches:  matches,
	}
}
