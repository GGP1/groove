import { Comment } from "./comment";
import { Event } from "./event";
import { Notification } from "./notification";
import { Post } from "./post";
import { Product } from "./product";
import { User } from "./user";

export type ErrResponse = {
	status: number
	error: string
}

export type IDResponse = {
	id: string
}

export type NameResponse = {
	name: string
}

export type UserIDBody = {
	user_id: string
}

export type BlockedIDBody = {
	blocked_id: string
}

export type FriendIDBody = {
	friend_id: string
}

interface Pagination {
	next_cursor: string
}

export interface EventsPag extends Pagination {
	events: Event[]
}

export interface UsersPag extends Pagination {
	users: User[]
}

export interface PostsPag extends Pagination {
	posts: Post[]
}

export interface CommentsPag extends Pagination {
	comments: Comment[]
}

export interface ProductsPag extends Pagination {
	products: Product[]
}

export interface NotificationsPag extends Pagination {
	notifications: Notification[]
}

interface Count {
	status: number
}

export interface AttendingEventsCount extends Count {
	attending_events_count: number
}

export interface BannedCount extends Count {
	banned_count: number
}

export interface BannedEventsCount extends Count {
	banned_events_count: number
}

export interface BannedFriendsCount extends Count {
	banned_friends_count: number
}

export interface BlockedCount extends Count {
	blocked_count: number
}

export interface BlockedByCount extends Count {
	blocked_by_count: number
}

export interface FollowersCount extends Count {
	followers_count: number
}

export interface FollowingCount extends Count {
	following_count: number
}

export interface FriendsCount extends Count {
	friends_count: number
}

export interface FriendsInCommonCount extends Count {
	friends_in_common_count: number
}

export interface FriendsNotInCommonCount extends Count {
	friends_not_in_common_count: number
}

export interface HostedEventsCount extends Count {
	hosted_events_count: number
}

export interface InvitedCount extends Count {
	invited_count: number
}

export interface InvitedEventsCount extends Count {
	invited_events_count: number
}

export interface InvitedFriendsCount extends Count {
	invited_friends_count: number
}

export interface LikesCount extends Count {
	likes_count: number
}

export interface LikedEventsCount extends Count {
	liked_events_count: number
}

export interface LikedByFriendsCount extends Count {
	liked_by_friends_count: number
}

export interface MembersCount extends Count {
	members_count: number
}

export interface MembersFriendsCount extends Count {
	members_friends_count: number
}

export interface NotificationsCount extends Count {
	notifications_count: number
}

export interface AvailableTickets extends Count {
	available_tickets_count: number
}

export interface CommentLikes extends Count {
	comment_likes_count: number
}

export interface PostLikes extends Count {
	post_likes_count: number
}
