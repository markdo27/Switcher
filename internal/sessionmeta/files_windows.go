package sessionmeta

import "context"

const (
	maxFile    = 1 << 20
	maxBytes   = 32 << 20
	maxEntries = 4096
)

type budget struct {
	ctx     context.Context
	entries int
	bytes   int64
}

func newBudget(ctx context.Context) *budget {
	return &budget{ctx: ctx, entries: maxEntries, bytes: maxBytes}
}

func (b *budget) active() bool {
	return b.ctx.Err() == nil && b.entries > 0 && b.bytes > 0
}

// Fail closed until metadata traversal has a Windows security backend.
func walkMetadata(string, int, *budget, func(string) bool, func(string, []byte), func(string)) bool {
	return false
}
