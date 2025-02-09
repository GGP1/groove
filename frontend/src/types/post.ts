
export type Post = {
	id: string,
	event_id: string,
	content: string,
	likes_count: number,
	comments_count: number,
	auth_user_liked: boolean,
	media: string[],
	created_at: Date,
	updated_at?: Date
}

export type CreatePost = {
	content: string,
	media: string[],
}

export type UpdatePost = {
	content?: string,
	likes_delta?: number
}
