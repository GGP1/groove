package event

// Predicate
const (
	Banned    predicate = "banned"
	Confirmed predicate = "confirmed"
	Invited   predicate = "invited"
	LikedBy   predicate = "liked_by"
)

// Query
const (
	_ query = iota
	banned
	bannedCount
	bannedLookup
	confirmed
	confirmedLookup
	confirmedCount
	invited
	invitedLookup
	invitedCount
	likedBy
	likedByLookup
	likedByCount
)

type predicate string
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
	confirmed: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(event_id, $id)) {
			confirmed (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	confirmedCount: `query q($id: string) {
		q(func: eq(event_id, $id)) {
			count(confirmed)
		}
	}`,
	confirmedLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(event_id, $id)) {
			confirmed @filter(eq(user_id, $lookup_id)) {
				user_id
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

// TODO: add counts (invitedFriendsCount, etc)?
const (
	// get users banned from the event that a friend of user
	bannedFriends mixedQuery = iota + 1
	// get users confirmed in the event that a friend of user
	confirmedFriends
	// get users invited to the event that a friend of user
	invitedFriends
	// get users liking the event that a friend of user
	likedByFriends
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
	confirmedFriends: `query q($event_id: string, $user_id: string, $cursor: string, $limit: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~confirmed, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
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
	likedByFriends: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			friend @filter(uid_in(~liked_by, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
}
