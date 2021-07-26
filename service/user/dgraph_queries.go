package user

// Query
const (
	_ query = iota
	banned
	bannedCount
	bannedLookup
	blocked
	blockedCount
	blockedLookup
	blockedBy
	blockedByCount
	blockedByLookup
	confirmed
	confirmedLookup
	confirmedCount
	friends
	friendsLookup
	friendsCount
	invited
	invitedLookup
	invitedCount
	likedBy
	likedByLookup
	likedByCount
)

type query uint8

// getQuery contains dgraphs queries to get user edges.
var getQuery = map[query]string{
	banned: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			~banned (orderasc: event_id) (first: $limit, offset: $cursor) {
				event_id
			}
		}
	}`,
	bannedCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(~banned)
		}
	}`,
	bannedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			~banned @filter(eq(event_id, $lookup_id)) {
				event_id
			}
		}
	}`,
	blocked: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			blocked (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	blockedCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(blocked)
		}
	}`,
	blockedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			blocked @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	blockedBy: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			~blocked (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	blockedByCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(~blocked)
		}
	}`,
	blockedByLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			~blocked @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	confirmed: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			~confirmed (orderasc: event_id) (first: $limit, offset: $cursor) {
				event_id
			}
		}
	}`,
	confirmedCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(~confirmed)
		}
	}`,
	confirmedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			~confirmed @filter(eq(event_id, $lookup_id)) {
				event_id
			}
		}
	}`,
	friends: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			friend (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	friendsCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(friend)
		}
	}`,
	friendsLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			friend @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	invited: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			~invited (orderasc: event_id) (first: $limit, offset: $cursor) {
				event_id
			}
		}
	}`,
	invitedCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(~invited)
		}
	}`,
	invitedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			~invited @filter(eq(event_id, $lookup_id)) {
				event_id
			}
		}
	}`,
	likedBy: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			~liked_by (orderasc: event_id) (first: $limit, offset: $cursor) {
				event_id
			}
		}
	}`,
	likedByCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(~liked_by)
		}
	}`,
	likedByLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			~liked_by @filter(eq(event_id, $lookup_id)) {
				event_id
			}
		}
	}`,
}

const (
	// Blocked queries are not implemented as they information is irrelevant for the user that blocked them.
	_ mixedQuery = iota
	// get users friends of user that are friends of target as well
	friendsOfFriend
	friendsOfFriendLookup
)

// mixedQuery looks for matches in two predicates (one from a user and one from an event) instead of one.
type mixedQuery uint8

// getMixedQuery is a list with queries that check two predicates.
var getMixedQuery = map[mixedQuery]string{
	friendsOfFriend: `query q($user_id: string, $target_user_id: string) {
		target as var(func: eq(user_id, $target_user_id))

		q(func: eq(user_id, $user_id)) {
			friend @filter(uid_in(friend, uid(target))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	friendsOfFriendLookup: `query q($user_id: string, $target_user_id: string, $lookup_id: string) {
		target as var(func: eq(user_id, $target_user_id))

		q(func: eq(user_id, $user_id)) {
			friend @filter(uid_in(friend, uid(target)) AND (eq(user_id, $lookup_id))) {
				user_id
			}
		}
	}`,
}
