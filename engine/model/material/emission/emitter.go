package emission

import (
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

type Emitter interface {
	Emit(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum // Evaluates emitted radiance along the outgoing direction.
	IsDelta() bool                                                    // Reports whether emission is delta-distributed.
}
