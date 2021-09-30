package user

// Query
const (
	banned query = iota + 1
	bannedCount
	bannedLookup
	blocked
	blockedCount
	blockedLookup
	blockedBy
	blockedByCount
	blockedByLookup
	friends
	friendsLookup
	friendsCount
	areFriends
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
	areFriends: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			friend @filter(eq(user_id, $lookup_id)) {
				count(user_id)
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
	friendsInCommon mixedQuery = iota + 1
	friendsInCommonCount
	friendsInCommonLookup
	friendsNotInCommon
	friendsNotInCommonCount
	friendsNotInCommonLookup
)

// mixedQuery looks for matches in two predicates (one from a user and one from an event) instead of one.
type mixedQuery uint8

// getMixedQuery is a list with queries that check two predicates.
var getMixedQuery = map[mixedQuery]string{
	friendsInCommon: `query q($id: string, $friend_id: string, $cursor: string, $limit: string) {
		user as var(func: eq(user_id, $id))
		target as var(func: eq(user_id, $friend_id))

		q(func: uid(user)) {
			friend @filter(uid_in(friend, uid(target)) AND uid_in(friend, uid(user))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	friendsInCommonCount: `query q($id: string, $friend_id: string) {
		user as var(func: eq(user_id, $id))
		target as var(func: eq(user_id, $friend_id))

		q(func: uid(user)) {
			count(friend) @filter(uid_in(friend, uid(target)) AND uid_in(friend, uid(user)))
		}
	}`,
	friendsInCommonLookup: `query q($id: string, $friend_id: string, $lookup_id: string) {
		user as var(func: eq(user_id, $id))
		target as var(func: eq(user_id, $friend_id))

		q(func: uid(user)) {
			friend @filter(uid_in(friend, uid(target)) AND uid_in(friend, uid(user)) AND eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	friendsNotInCommon: `query q($id: string, $friend_id: string, $cursor: string, $limit: string) {
		user as var(func: eq(user_id, $id))
		target as var(func: eq(user_id, $friend_id))

		q(func: uid(user)) {
			friend @filter((NOT uid_in(friend, uid(target))) AND (NOT uid(target))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	friendsNotInCommonCount: `query q($id: string, $friend_id: string) {
		user as var(func: eq(user_id, $id))
		target as var(func: eq(user_id, $friend_id))

		q(func: uid(user)) {
			count(friend) @filter((NOT uid_in(friend, uid(target))) AND (NOT uid(target)))
		}
	}`,
	friendsNotInCommonLookup: `query q($id: string, $friend_id: string, $lookup_id: string) {
		user as var(func: eq(user_id, $id))
		target as var(func: eq(user_id, $friend_id))

		q(func: uid(user)) {
			friend @filter((NOT uid_in(friend, uid(target))) AND (NOT uid(target)) AND  eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
}
