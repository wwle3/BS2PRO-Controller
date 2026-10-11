package main

import (
	"github.com/TIANLI0/BS2PRO-Controller/internal/ipc"
	"github.com/TIANLI0/BS2PRO-Controller/internal/types"
)

func (a *CoreApp) setRuntimeFanCurve(curve []types.FanCurvePoint) bool {
	return a.storeRuntimeFanCurve(curve, 0, false)
}

func (a *CoreApp) setRuntimeFanCurveIfGen(curve []types.FanCurvePoint, gen uint64) bool {
	return a.storeRuntimeFanCurve(curve, gen, true)
}

func (a *CoreApp) storeRuntimeFanCurve(curve []types.FanCurvePoint, gen uint64, matchGen bool) bool {
	clone := cloneFanCurvePoints(curve)
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if matchGen && gen != a.runtimeCurveGen {
		return false
	}
	if fanCurvesEqual(a.runtimeFanCurve, clone) {
		return false
	}
	a.runtimeFanCurve = clone
	a.runtimeCurveGen++
	return true
}

func (a *CoreApp) clearRuntimeFanCurve() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.clearRuntimeFanCurveLocked()
}

func (a *CoreApp) clearRuntimeFanCurveLocked() {
	a.runtimeCurveGen++
	if len(a.runtimeFanCurve) == 0 {
		return
	}
	a.runtimeFanCurve = nil
}

func (a *CoreApp) visibleFanCurve(base []types.FanCurvePoint) []types.FanCurvePoint {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	if len(a.runtimeFanCurve) > 0 {
		return cloneFanCurvePoints(a.runtimeFanCurve)
	}
	return cloneFanCurvePoints(base)
}

func (a *CoreApp) broadcastConfigUpdate(cfg types.AppConfig) {
	if a.ipcServer == nil {
		return
	}
	a.ipcServer.BroadcastEvent(ipc.EventConfigUpdate, cfg)
}

func (a *CoreApp) restoreLearnedFanCurve(base []types.FanCurvePoint) {
	if a == nil || a.fanOffsetCtrl == nil || len(base) == 0 {
		return
	}
	restored, changed := a.fanOffsetCtrl.MaterializeLearnedCurve(base, 500, 4000)
	if !changed {
		return
	}
	a.setRuntimeFanCurve(restored)
	a.logInfo("已按长期记忆恢复整条风扇偏移曲线", "points", len(restored))
}

func (a *CoreApp) broadcastVisibleFanCurve(base []types.FanCurvePoint) {
	if a.ipcServer == nil {
		return
	}
	a.ipcServer.BroadcastEvent(ipc.EventFanCurveUpdate, a.visibleFanCurve(base))
}

func fanCurvesEqual(a, b []types.FanCurvePoint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Temperature != b[i].Temperature || a[i].RPM != b[i].RPM || a[i].Offset != b[i].Offset {
			return false
		}
	}
	return true
}
