package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func handleMember(source mautrix.EventSource, evt *event.Event) {
	content := evt.Content.AsMember()
	prevMembership := event.MembershipLeave
	if evt.Unsigned.PrevContent != nil {
		prevMembership = evt.Unsigned.PrevContent.AsMember().Membership
	}
	log := globalLog.With().
		Str("event_id", evt.ID.String()).
		Str("action", "membership event").
		Logger()
	ctx := log.WithContext(context.WithValue(context.Background(), contextKeyEvent, evt))
	log.Debug().
		Str("room_id", evt.RoomID.String()).
		Str("sender", evt.Sender.String()).
		Str("state_key", evt.GetStateKey()).
		Str("membership", string(content.Membership)).
		Str("prev_membership", string(prevMembership)).
		Str("source", source.String()).
		Msg("Received member event")
	if evt.GetStateKey() == cli.UserID.String() {
		if content.Membership != event.MembershipInvite || source&mautrix.EventSourceInvite == 0 {
			log.Debug().Msg("Ignoring non-invite member event for self")
		} else if evt.Sender.Homeserver() != cli.UserID.Homeserver() {
			log.Debug().Msg("Rejecting invite from user on different homeserver")
			leaveRoom(ctx, evt.RoomID, fmt.Sprintf("This bot only serves users on %s", cli.UserID.Homeserver()))
		} else if stateProblem := inviteLooksPrivate(evt); stateProblem != "" {
			log.Debug().
				Str("problem", stateProblem).
				Msg("Rejecting invite to room that doesn't look like a private chat")
			leaveRoom(ctx, evt.RoomID, "This bot only accepts invites to encrypted direct chats")
		} else {
			log.Debug().Msg("Accepting direct chat invite")
			acceptInvite(ctx, evt)
		}
	} else if source&mautrix.EventSourceJoin > 0 {
		_, err := getOtherUserID(ctx, evt.RoomID, false, true)
		if errors.Is(err, errWrongMemberCount) {
			log.Debug().Msg("Room has more than 2 members now, leaving")
			leaveRoom(ctx, evt.RoomID, "Another user joined the room")
		} else if err != nil {
			log.Err(err).Msg("Failed to get members in room after member event from someone else")
		}
	} else {
		log.Debug().Msg("Ignoring state event in invite stream")
	}
}

func leaveRoom(ctx context.Context, roomID id.RoomID, reason string) {
	_, err := cli.LeaveRoom(roomID, &mautrix.ReqLeave{Reason: reason})
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("reason", reason).Msg("Failed to leave room")
	}
}

func acceptInvite(ctx context.Context, evt *event.Event) {
	log := zerolog.Ctx(ctx)
	_, err := cli.JoinRoomByID(evt.RoomID)
	if err != nil {
		log.Err(err).Msg("Failed to accept invite")
		return
	}
	_, err = getOtherUserID(ctx, evt.RoomID, false, false)
	if errors.Is(err, errWrongMemberCount) {
		log.Debug().Msg("Room has more than 2 members after accepting invite, leaving")
		leaveRoom(ctx, evt.RoomID, "This bot only accepts direct chat invites")
	} else if err != nil {
		log.Err(err).Msg("Failed to check members in room after accepting invite")
		leaveRoom(ctx, evt.RoomID, "Failed to check members in room")
	} else {
		log.Debug().Msg("Room only has 2 members, staying in room")
	}
}

func inviteLooksPrivate(evt *event.Event) string {
	if !evt.Content.AsMember().IsDirect {
		return "invite isn't flagged as direct"
	}
	isEncrypted := false
	for _, stateEvt := range evt.Unsigned.InviteRoomState {
		switch stateEvt.Type {
		case event.StateJoinRules:
			if stateEvt.Content.AsJoinRules().JoinRule != event.JoinRuleInvite {
				return "join rule is not invite"
			}
		case event.StateCreate:
			if stateEvt.Sender != evt.Sender {
				return "room was created by different user"
			}
		case event.StateRoomName:
			return "room has name"
		case event.StateRoomAvatar:
			return "room has avatar"
		case event.StateTopic:
			return "room has topic"
		case event.StateCanonicalAlias:
			return "room has canonical alias"
		case event.StateEncryption:
			if stateEvt.Content.AsEncryption().Algorithm != id.AlgorithmMegolmV1 {
				return "room has unknown encryption algorithm"
			}
			isEncrypted = true
		}
	}
	if !isEncrypted {
		return "room is not encrypted"
	}
	return ""
}

var errWrongMemberCount = errors.New("room has more than 2 members")

var memberValidatedRooms = make(map[id.RoomID]id.UserID)
var memberValidatedRoomsLock sync.Mutex

func getOtherUserID(ctx context.Context, roomID id.RoomID, allowMemoryCache, allowDBCache bool) (id.UserID, error) {
	memberValidatedRoomsLock.Lock()
	defer memberValidatedRoomsLock.Unlock()
	if allowMemoryCache {
		if userID, ok := memberValidatedRooms[roomID]; ok {
			return userID, nil
		}
	}
	log := zerolog.Ctx(ctx)
	members, err := cli.StateStore.GetRoomJoinedOrInvitedMembers(roomID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch members from cache: %w", err)
	}
	cached := true
	if len(members) == 0 || !allowDBCache {
		cached = false
		log.Debug().Msg("No members in cache, fetching from server")
		joinedMembers, err := cli.JoinedMembers(roomID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch members from server: %w", err)
		}
		members = make([]id.UserID, 0, len(joinedMembers.Joined))
		for userID := range joinedMembers.Joined {
			members = append(members, userID)
		}
	}
	log.Debug().Interface("members", members).Bool("from_cache", cached).Msg("Got members in room")
	if len(members) != 2 {
		memberValidatedRooms[roomID] = ""
		return "", errWrongMemberCount
	}
	var otherUserID id.UserID
	if members[0] == cli.UserID {
		otherUserID = members[1]
	} else if members[1] == cli.UserID {
		otherUserID = members[0]
	} else {
		return "", fmt.Errorf("neither member in room is the bot")
	}
	memberValidatedRooms[roomID] = otherUserID
	return otherUserID, nil
}

// TODO move this to mautrix-go
func moveInviteState(resp *mautrix.RespSync, _ string) bool {
	for _, meta := range resp.Rooms.Invite {
		var inviteState []event.StrippedState
		var inviteEvt *event.Event
		for _, evt := range meta.State.Events {
			if evt.Type == event.StateMember && evt.GetStateKey() == cli.UserID.String() {
				inviteEvt = evt
			} else {
				evt.Type.Class = event.StateEventType
				_ = evt.Content.ParseRaw(evt.Type)
				inviteState = append(inviteState, event.StrippedState{
					Content:  evt.Content,
					Type:     evt.Type,
					StateKey: evt.GetStateKey(),
					Sender:   evt.Sender,
				})
			}
		}
		if inviteEvt != nil {
			inviteEvt.Unsigned.InviteRoomState = inviteState
			meta.State.Events = []*event.Event{inviteEvt}
		}
	}
	return true
}
