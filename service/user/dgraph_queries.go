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
	followedBy
	followedByLookup
	followedByCount
	following
	followingLookup
	followingCount
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
	followedBy: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			~following (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	followedByCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(~following)
		}
	}`,
	followedByLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			~following @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	following: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(user_id, $id)) {
			following (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	followingCount: `query q($id: string) {
		q(func: eq(user_id, $id)) {
			count(following)
		}
	}`,
	followingLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(user_id, $id)) {
			following @filter(eq(user_id, $lookup_id)) {
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

// Blocked queries are not implemented as they information is irrelevant for the user that blocked them.
const (
	_ mixedQuery = iota
	// get users following user that follow target
	followersFollowing
	followersFollowingLookup
	// get users followed by user that follow target
	followingFollowing
	followingFollowingLookup
	// get users followed by both user and target
	followingFollowers
	followingFollowersLookup
)

// mixedQuery looks for matches in two predicates (one from a user and one from an event) instead of one.
type mixedQuery uint8

// getMixedQuery is a list with queries that check two predicates.
var getMixedQuery = map[mixedQuery]string{
	followersFollowing: `query q($user_id: string, $target_user_id: string) {
		target as var(func: eq(user_id, $target_user_id))
		
		q(func: eq(user_id, $user_id)) {
			~following @filter(uid_in(following, uid(target))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	followersFollowingLookup: `query q($user_id: string, $target_user_id: string, $lookup_id: string) {
		target as var(func: eq(user_id, $target_user_id))
		
		q(func: eq(user_id, $user_id)) {
			~following @filter(uid_in(following, uid(target)) AND (eq(user_id, $lookup_id))) {
				user_id
			}
		}
	}`,
	followingFollowing: `query q($user_id: string, $target_user_id: string) {
		target as var(func: eq(user_id, $target_user_id))

		q(func: eq(user_id, $user_id)) {
			following @filter(uid_in(following, uid(target))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	followingFollowingLookup: `query q($user_id: string, $target_user_id: string, $lookup_id: string) {
		target as var(func: eq(user_id, $target_user_id))

		q(func: eq(user_id, $user_id)) {
			following @filter(uid_in(following, uid(target)) AND (eq(user_id, $lookup_id))) {
				user_id
			}
		}
	}`,
	followingFollowers: `query q($user_id: string, $target_user_id: string) {
		target as var(func: eq(user_id, $target_user_id))
		
		q(func: eq(user_id, $user_id)) {
			following @filter(uid_in(~following, uid(target))) (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	followingFollowersLookup: `query q($user_id: string, $target_user_id: string, $lookup_id: string) {
		target as var(func: eq(user_id, $target_user_id))
		
		q(func: eq(user_id, $user_id)) {
			following @filter(uid_in(~following, uid(target)) AND (eq(user_id, $lookup_id))) {
				user_id
			}
		}
	}`,
}
