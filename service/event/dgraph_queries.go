package event

import (
	"github.com/GGP1/groove/internal/params"
)

// Query
const (
	banned query = iota + 1
	bannedCount
	bannedLookup
	isBanned
	invited
	invitedLookup
	invitedCount
	isInvited
	likedBy
	likedByLookup
	likedByCount
)

type query uint8

// getQuery contains dgraphs queries to get event edges.
var getQuery = map[query]string{
	banned: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(event_id, $id)) {
			banned (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	bannedCount: `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(banned)
		}
	}`,
	bannedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(event_id, $id)) {
			banned @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	isBanned: `query q($id: string, $lookup_id: string) {
		q(func: eq(event_id, $id)) {
			banned @filter(eq(user_id, $lookup_id)) {
				count(user_id)
			}
		}
	}`,
	invited: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(event_id, $id)) {
			invited (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	invitedCount: `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(invited)
		}
	}`,
	invitedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(event_id, $id)) {
			invited @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	isInvited: `query q($id: string, $lookup_id: string) {
		q(func: eq(event_id, $id)) {
			invited @filter(eq(user_id, $lookup_id)) {
				count(user_id)
			}
		}
	}`,
	likedBy: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(event_id, $id)) {
			liked_by (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	likedByCount: `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(liked_by)
		}
	}`,
	likedByLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(event_id, $id)) {
			liked_by @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
}

const (
	// get users banned from the event that are friend of user
	bannedFriends mixedQuery = iota + 1
	bannedFriendsCount
	bannedFriendsLookup
	// get users invited to the event that are friend of user
	invitedFriends
	invitedFriendsCount
	invitedFriendsLookup
	// get users liking the event that are friend of user
	likedByFriends
	likedByFriendsCount
	likedByFriendsLookup
)

// mixedQuery looks for matches in two predicates (one from a user and one from an event) instead of one.
type mixedQuery uint8

// getMixedQuery is a list with queries that check two predicates.
var getMixedQuery = map[mixedQuery]string{
	bannedFriends: `query q($event_id: string, $user_id: string, $cursor: string, $limit: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~banned, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	bannedFriendsCount: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			count(friend) @filter(uid_in(~banned, uid(event)))
		}
	}`,
	bannedFriendsLookup: `query q($event_id: string, $user_id: string, $lookup_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~banned, uid(event)) AND eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	invitedFriends: `query q($event_id: string, $user_id: string, $cursor: string, $limit: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~invited, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	invitedFriendsCount: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			count(friend) @filter(uid_in(~invited, uid(event)))
		}
	}`,
	invitedFriendsLookup: `query q($event_id: string, $user_id: string, $lookup_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~invited, uid(event)) AND eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	likedByFriends: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~liked_by, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	likedByFriendsCount: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			count(friend) @filter(uid_in(~liked_by, uid(event)))
		}
	}`,
	likedByFriendsLookup: `query q($event_id: string, $user_id: string, $lookup_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~liked_by, uid(event)) AND eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
}

func mixedQueryVars(eventID, userID string, params params.Query) map[string]string {
	vars := map[string]string{
		"$event_id": eventID,
		"$user_id":  userID,
	}
	if params.LookupID != "" {
		vars["$lookup_id"] = params.LookupID
		return vars
	}

	vars["$cursor"] = params.Cursor
	vars["$limit"] = params.Limit

	return vars
}
