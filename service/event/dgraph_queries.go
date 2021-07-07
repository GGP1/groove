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
	banned query = iota
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

// Following queries require users to follow each other
const (
	_ mixedQuery = iota
	// get users banned from the event that are followers of user
	bannedFollowers
	// get users banned from the event that are followed by user
	bannedFollowing
	// get users confirmed in the event that are followers of user
	confirmedFollowers
	// get users confirmed in the event that are followed by user
	confirmedFollowing
	// get users invited to the event that are followers of user
	invitedFollowers
	// get users invited to the event that are followed by user
	invitedFollowing
	// get users liking the event that are followers of user
	likedByFollowers
	// get users liking the event that are followed by user
	likedByFollowing
)

// mixedQuery looks for matches in two predicates (one from a user and one from an event) instead of one.
type mixedQuery uint8

// getMixedQuery is a list with queries that check two predicates.
// TODO: add lookup queries?
var getMixedQuery = map[mixedQuery]string{
	bannedFollowers: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			~following @filter(uid_in(~banned, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	bannedFollowing: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			following @filter(uid_in(~banned, uid(event)) AND uid_in(following, uid(user))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	confirmedFollowers: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			~following @filter(uid_in(~confirmed, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	confirmedFollowing: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			following @filter(uid_in(~confirmed, uid(event)) AND uid_in(following, uid(user))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	invitedFollowers: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			~following @filter(uid_in(~invited, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	invitedFollowing: `query q($event_id: string, $user_id: string, $cursor: string, $limit: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			following @filter(uid_in(~invited, uid(event)) AND uid_in(following, uid(user))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	likedByFollowers: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			~following @filter(uid_in(~liked_by, uid(event))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	likedByFollowing: `query q($event_id: string, $user_id: string) {
		user as var(func: eq(user_id, $user_id))
		event as var(func: eq(event_id, $event_id))

		q(func: uid(user)) {
			following @filter(uid_in(~liked_by, uid(event)) AND uid_in(following, uid(user))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
}
