package main

import (
	"context"
	"sync"
)

type tPingers struct {
	sync.Mutex
	list map[uint16]*tPingersEntry
}
type tPingersEntry struct {
	ctxStartWait         context.Context
	ctxStartWaitDoneFunc context.CancelFunc
	entry                *tPingerWrap
}

func (thisPingers *tPingers) addPinger(id uint16, pinger *tPingersEntry) {
	thisPingers.Lock()
	defer thisPingers.Unlock()

	thisPingers.list[id] = pinger
}

func (thisPingers *tPingers) getPinger(id uint16) (*tPingersEntry, bool) {
	thisPingers.Lock()
	defer thisPingers.Unlock()

	pinger, ok := thisPingers.list[id]

	return pinger, ok
}

func (thisPingers *tPingers) deletePinger(id uint16) {
	thisPingers.Lock()
	defer thisPingers.Unlock()

	delete(thisPingers.list, id)
}
