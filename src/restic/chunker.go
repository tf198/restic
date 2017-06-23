package restic

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	"restic/errors"
	"restic/pool"

	"github.com/restic/chunker"
)

func ChunkHash(ids []ID) string {
	w := sha256.New()
	for _, id := range ids {
		w.Write(id[:])
	}
	return hex.EncodeToString(w.Sum(nil))
}

type chunkOffsets struct {
	Start  int64
	Length int64
}

type chunkResult struct {
	id      ID
	offsets chunkOffsets
}

func FileChunks(rd io.Reader, pol chunker.Pol) (map[string]chunkOffsets, string, error) {

	chnker := chunker.New(rd, pol)
	resultChannels := [](<-chan chunkResult){}

	for {
		chunk, err := chnker.Next(pool.GetBuf())
		if errors.Cause(err) == io.EOF {
			break
		}

		if err != nil {
			return nil, "", errors.Wrap(err, "chunker.Next")
		}

		resCh := make(chan chunkResult, 1)
		go func() {
			defer pool.FreeBuf(chunk.Data)
			id := Hash(chunk.Data)
			resCh <- chunkResult{id: id, offsets: chunkOffsets{int64(chunk.Start), int64(chunk.Length)}}
		}()
		resultChannels = append(resultChannels, resCh)
	}

	results := make(map[string]chunkOffsets)

	w := sha256.New()

	for _, ch := range resultChannels {
		result := <-ch
		w.Write(result.id[:])
		results[result.id.String()] = result.offsets
	}

	return results, hex.EncodeToString(w.Sum(nil)), nil

}
