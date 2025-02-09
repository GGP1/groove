import { useNavigation } from "@react-navigation/native";
import React, { useEffect, useState } from "react";
import { Dimensions, Pressable, StyleSheet, Text, View } from "react-native";
import { Avatar, Divider } from "react-native-elements";
import { CommentItem, List, SocialCounts } from "..";
import { API, Params } from "../../api/api";
import { UsersPag } from "../../types/api";
import { Comment } from "../../types/comment";
import { User } from "../../types/user";
import { formatPostDate } from "../../utils/date";
import { HTTP } from "../../utils/http";
import { HomeNavigationProps } from "../../utils/navigation";
import { FetchItemsResponse } from "../List";
import { Reply } from "./Reply";

interface Props {
	comment_id: string,
	event_id: string,
	user_id: string,
}

const { width } = Dimensions.get("screen");

// TODO: try to implement a single component for both post and comment appearing as main content

/** Commentx represents a post comment or a comment reply. */
export const Commentx = (props: Props) => {
	const [comment, setComment] = useState<Comment>();
	const [user, setUser] = useState<User>();
	const [repliesCount, setRepliesCount] = useState<number>(0);
	const [refreshReplies, setRefreshReplies] = useState<boolean>(false);
	const navigation = useNavigation<HomeNavigationProps>();

	useEffect(() => {
		const getComment = async () => {
			try {
				const c = await API.Events.GetComment(props.event_id, props.comment_id);
				setComment(c);
			} catch (err) {
				console.log(err);
			}
		};
		const getUser = async () => {
			try {
				const u = await API.Users.GetByID(props.user_id);
				setUser(u);
			} catch (err) {
				console.log(err);
			}
		};

		HTTP.paralell([getComment(), getUser()]);
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [refreshReplies]);

	const getReplies = async (params?: Params<Comment>): FetchItemsResponse<Comment> => {
		const resp = await API.Events.GetReplies(props.event_id, props.comment_id, params);
		setRepliesCount(resp.comments ? repliesCount + resp.comments.length : repliesCount);
		return [resp.next_cursor, resp.comments];
	};

	const getLikes = async (params?: Params<User>): FetchItemsResponse<User> => {
		const resp = await API.Events.GetCommentLikes<UsersPag>(props.event_id, props.comment_id, params);
		return [resp.next_cursor, resp.users];
	};

	const goToUser = () => navigation.navigate("User", { user_id: props.user_id, item: user });

	const Commentt = () => (
		<View style={styles.commentContainer}>
			<View style={styles.header}>
				<Pressable onPress={goToUser}>
					<Avatar rounded size={45} source={{ uri: user?.profile_image_url }} />
				</Pressable>
				<View style={styles.title}>
					<Pressable onPress={goToUser}>
						<Text style={styles.name}>{user?.username}</Text>
					</Pressable>
				</View>
			</View>

			<Text style={styles.content}>{comment?.content}</Text>
			<Text style={styles.createdAt}>{formatPostDate(comment ? comment.created_at : new Date())}</Text>

			<Divider width={0.8} style={styles.divider} />

			{comment &&
				<View>
					<SocialCounts
						likes_count={comment.likes_count}
						comment_count={comment.replies_count}
						liked={comment.auth_user_liked}
						like={() => API.Events.LikeComment(props.event_id, comment.id)}
						getUsersLikes={(params) => getLikes(params)}
					/>
					<Divider style={styles.divider} />
					<Reply
						event_id={props.event_id}
						// Even though we have the post id we don't pass it so the new comment is saved
						// as the child of this one
						parent_comment_id={comment.id}
						onSubmit={() => setRefreshReplies(!refreshReplies)}
					/>
				</View>
			}
		</View>
	);

	return (
		<View style={styles.container}>
			<List<Comment>
				fetchItems={(params) => getReplies(params)}
				renderItem={({ item }) => <CommentItem event_id={props.event_id} comment={item} />}
				ListHeaderComponent={Commentt}
				extraData={refreshReplies}
			/>
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		backgroundColor: "white",
	},
	commentContainer: {
		margin: 15,
		marginBottom: 10,
	},
	header: {
		flexDirection: "row",
		alignItems: "center",
		marginBottom: 10,
	},
	title: {
		marginLeft: 10,
	},
	name: {
		fontSize: 18,
		fontWeight: "bold",
	},
	content: {
		fontSize: 17,
	},
	createdAt: {
		fontSize: 15,
		marginTop: 10,
		color: "rgba(0, 0, 0, 0.6)",
		textAlign: "right",
	},
	divider: {
		marginVertical: 12,
		marginLeft: -15, // Counter postContainer margin
		width: width,
	},
});
