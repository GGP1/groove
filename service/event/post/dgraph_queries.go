package post

import "github.com/dgraph-io/dgo/v210/protos/api"

const (
	mutateCommentLikes query = iota + 1
	mutatePostLikes
	commentLikes
	commentLikesCount
	commentLikesLookup
	postLikes
	postLikesCount
	postLikesLookup
	removeCommentLike
	removePostLike
)

type query uint8

var queries = map[query]string{
	mutateCommentLikes: `query q($comment_id: string, $user_id: string) {
		comment as var(func: eq(comment_id, $comment_id))
		user as var(func: eq(user_id, $user_id))
	}`,
	mutatePostLikes: `query q($post_id: string, $user_id: string) {
		post as var(func: eq(post_id, $post_id))
		user as var(func: eq(user_id, $user_id))
	}`,
	commentLikes: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(comment_id, $id)) {
			liked_by (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	commentLikesCount: `query q($id: string) {
		q(func: eq(comment_id, $id)) {
			count(liked_by)
		}
	}`,
	commentLikesLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(comment_id, $id)) {
			liked_by @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
	postLikes: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(post_id, $id)) {
			liked_by (orderasc: user_id) (first: $limit, offset: $cursor) {
				user_id
			}
		}
	}`,
	postLikesCount: `query q($id: string, $cursor: string, $limit: string) {
		q(func: eq(post_id, $id)) {
			count(liked_by)
		}
	}`,
	postLikesLookup: `query q($id: string, $lookup_id: string) {
		q(func: eq(post_id, $id)) {
			liked_by @filter(eq(user_id, $lookup_id)) {
				user_id
			}
		}
	}`,
}

func commentMutationReq(commentID, userID string, set bool) *api.Request {
	mu := &api.Mutation{
		Cond: "@if(eq(len(comment), 1) AND eq(len(user), 1))",
	}
	nQuads := []byte("uid(comment) <liked_by> uid(user) .")
	if set {
		mu.SetNquads = nQuads
	} else {
		mu.DelNquads = nQuads
	}
	return &api.Request{
		Vars:      map[string]string{"$comment_id": commentID, "$user_id": userID},
		Query:     queries[mutateCommentLikes],
		Mutations: []*api.Mutation{mu},
	}
}

func postMutationReq(postID, userID string, set bool) *api.Request {
	mu := &api.Mutation{
		Cond: "@if(eq(len(post), 1) AND eq(len(user), 1))",
	}
	nQuads := []byte("uid(post) <liked_by> uid(user) .")
	if set {
		mu.SetNquads = nQuads
	} else {
		mu.DelNquads = nQuads
	}
	return &api.Request{
		Vars:      map[string]string{"$post_id": postID, "$user_id": userID},
		Query:     queries[mutatePostLikes],
		Mutations: []*api.Mutation{mu},
	}
}
