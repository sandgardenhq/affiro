package oakx

import (
	"sync"

	"github.com/oakmound/oak/v4"
	"github.com/oakmound/oak/v4/alg/floatgeom"
	"github.com/oakmound/oak/v4/entities"
	"github.com/oakmound/oak/v4/event"
	"github.com/oakmound/oak/v4/mouse"
	"github.com/oakmound/oak/v4/scene"
)

var touchingMapLock sync.Mutex
var touchingMap = map[event.CallerID]bool{}

func RelativePhaseCollisionEnter(ctx *scene.Context) func(_ *entities.Entity, _ event.EnterPayload) event.Response {
	return func(e *entities.Entity, _ event.EnterPayload) event.Response {
		ev := mouse.LastEvent
		if ev.StopPropagation {
			return 0
		}
		vp := ctx.Window.(*oak.Window).Viewport()
		evSpace := ev.ToSpace()
		evSpace.Location.Min = evSpace.Location.Min.Add(floatgeom.Point3{float64(vp.X()), float64(vp.Y()), 0})
		evSpace.Location.Max = evSpace.Location.Max.Add(floatgeom.Point3{float64(vp.X()), float64(vp.Y()), 0})
		if e.Space.Contains(evSpace) {
			touchingMapLock.Lock()
			defer touchingMapLock.Unlock()
			if ok := touchingMap[e.CallerID]; !ok {
				touchingMap[e.CallerID] = true
				event.TriggerForCallerOn(ctx, e.CallerID, mouse.Start, &ev)
			}
		} else {
			touchingMapLock.Lock()
			defer touchingMapLock.Unlock()
			if ok := touchingMap[e.CallerID]; ok {
				touchingMap[e.CallerID] = false
				event.TriggerForCallerOn(ctx, e.CallerID, mouse.Stop, &ev)
			}
		}
		return 0
	}
}
