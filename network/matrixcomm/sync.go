package matrixcomm

import (
"encoding/json"
"fmt"
"runtime/debug"
"time"
)

// Syncer represents an interface that must be satisfied in order to do /sync requests on a client.
type Syncer interface {
	ProcessResponse(resp *RespSync, since string) error
	OnFailedSync(res *RespSync, err error) (time.Duration, error)
	GetFilterJSON(userID string) json.RawMessage
}

// DefaultSyncer is the default syncing implementation
type DefaultSyncer struct {
	UserID    string
	Store     Storer
	listeners map[string][]OnEventListener
}

// NewDefaultSyncer returns an instantiated DefaultSyncer
func NewDefaultSyncer(userID string, store Storer) *DefaultSyncer {
	return &DefaultSyncer{
		UserID:    userID,
		Store:     store,
		listeners: make(map[string][]OnEventListener),
	}
}

// notifyListeners as a callback and notify listener
func (s *DefaultSyncer) notifyListeners(event *Event) {
	tmpEventType:=event.Type
	listeners, exists := s.listeners[tmpEventType]
	if !exists {
		return
	}
	for _, fn := range listeners {
		fn(event)
	}
}

// OnEventListener can be used with DefaultSyncer.OnEventType to be informed of incoming events.
type OnEventListener func(x *Event)

// ProcessResponse 处理接收到的消息
func (s *DefaultSyncer) ProcessResponse(res *RespSync, since string) (err error) {
	if !s.shouldProcessResponse(res, since) {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessResponse panicked! userID=%s since=%s panic=%s\n%s", s.UserID, since, r, debug.Stack())
		}
	}()
	//消息中presence,获取presence
	for _,presenceUpdate:=range res.Presence.Events{
		s.notifyListeners(&presenceUpdate)
	}
	//消息中的AccountData，返回的是map address-roomid
	for _,event:=range res.AccountData.Events{
		s.notifyListeners(&event)
	}
	//消息中的room内发生的join事件
	for roomID, roomData := range res.Rooms.Join {
		room := s.getOrCreateRoom(roomID)
		for _, event := range roomData.State.Events {
			event.RoomID = roomID
			room.UpdateState(&event)
			s.notifyListeners(&event)
		}
		for _, event := range roomData.Timeline.Events {
			event.RoomID = roomID
			s.notifyListeners(&event)
		}
	}
	//消息中的room内发生的invite事件
	for roomID, roomData := range res.Rooms.Invite {
		room := s.getOrCreateRoom(roomID)
		for _, event := range roomData.State.Events {
			event.RoomID = roomID
			room.UpdateState(&event)
			s.notifyListeners(&event)
		}
	}
	//消息中的room内发生的leave事件
	for roomID, roomData := range res.Rooms.Leave {
		room := s.getOrCreateRoom(roomID)
		for _, event := range roomData.Timeline.Events {
			if event.StateKey != nil {
				event.RoomID = roomID
				room.UpdateState(&event)
				s.notifyListeners(&event)
			}
		}
	}
	return
}

// OnEventType allows callers to be notified when there are new events for the given event type.
// There are no duplicate checks.
func (s *DefaultSyncer) OnEventType(eventType string, callback OnEventListener) {
	_, exists := s.listeners[eventType]
	if !exists {
		s.listeners[eventType] = []OnEventListener{}
	}
	s.listeners[eventType] = append(s.listeners[eventType], callback)
}

// shouldProcessResponse returns true if the response should be processed.
// May modify the response to remove stuff that shouldn't be processed.
func (s *DefaultSyncer) shouldProcessResponse(resp *RespSync, since string) bool {
	if since == "" {
		return false
	}

	for roomID, roomData := range resp.Rooms.Join {
		for i := len(roomData.Timeline.Events) - 1; i >= 0; i-- {
			e := roomData.Timeline.Events[i]
			if e.Type == "m.room.member" && e.StateKey != nil && *e.StateKey == s.UserID {
				m := e.Content["membership"]
				mship, ok := m.(string)
				if !ok {
					continue
				}
				if mship == "join" {
					_, ok := resp.Rooms.Join[roomID]
					if !ok {
						continue
					}
					delete(resp.Rooms.Join, roomID)
					delete(resp.Rooms.Invite, roomID)
					break
				}
			}
		}
	}
	return true
}

// getOrCreateRoom must only be called by the Sync() goroutine which calls ProcessResponse()
func (s *DefaultSyncer) getOrCreateRoom(roomID string) *Room {
	room := s.Store.LoadRoom(roomID)
	if room == nil {
		room = NewRoom(roomID)
		s.Store.SaveRoom(room)
	}
	return room
}

// OnFailedSync always returns a 5 second wait period between failed /syncs, never a fatal error.
func (s *DefaultSyncer) OnFailedSync(res *RespSync, err error) (time.Duration, error) {
	return 2 * time.Second, nil
}

// GetFilterJSON returns a filter with a timeline limit of 50.
func (s *DefaultSyncer) GetFilterJSON(userID string) json.RawMessage {
	return json.RawMessage(`{"room":{"timeline":{"limit":50}}}`)
}
