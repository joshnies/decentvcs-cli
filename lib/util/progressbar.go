package util

import (
	"sync"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

// Create new progress bar with custom options.
func NewProgressBar(count int, name string) *mpb.Bar {
	p := mpb.New()
	return p.New(int64(count),
		mpb.BarStyle().Lbound("[").Filler("=").Tip(">").Padding("-").Rbound("]"),
		mpb.PrependDecorators(
			decor.Name(name, decor.WC{W: len(name) + 2, C: decor.DidentRight}),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
			decor.CountersNoUnit("(%d/%d)", decor.WCSyncSpace),
		),
	)
}

// Create new progress bar with custom options.
func NewProgressBarWithWaitGroup(barNames []string, wg *sync.WaitGroup) *mpb.Progress {
	p := mpb.New(mpb.WithWaitGroup(wg))
	for _, name := range barNames {
		p.AddBar(100,
			// mpb.BarStyle().Lbound("[").Filler("=").Tip(">").Padding("-").Rbound("]"),
			mpb.PrependDecorators(
				decor.Name(name, decor.WC{W: len(name) + 2, C: decor.DidentRight}),
			),
			mpb.AppendDecorators(
				decor.Percentage(),
				decor.CountersNoUnit("(%d/%d)", decor.WCSyncSpace),
			),
		)
	}

	return p
}
