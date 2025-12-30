package llm

import (
	"golang.org/x/sync/semaphore"
)

var (
	TextWeight  = int64(5)
	TextSem     = semaphore.NewWeighted(TextWeight)
	ImageWeight = int64(10)
	ImageSem    = semaphore.NewWeighted(ImageWeight)
	EmbedWeight = int64(50)
	EmbedSem    = semaphore.NewWeighted(EmbedWeight)
)
