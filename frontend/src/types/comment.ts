export type Comment = {
	id: string,
	parent_comment_id?: string,
	post_id?: string,
	user_id: string,
	content: string,
	likes_count: number,
	replies_count: number,
	auth_user_liked: boolean,
	created_at: Date,
}

export type CreateComment = {
	parent_comment_id?: string,
	post_id?: string,
	content: string,
}
