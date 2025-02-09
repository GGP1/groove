import { useNavigation } from "@react-navigation/core";
import React, { useEffect, useState } from "react";
import { Dimensions, StyleSheet, Text, View } from "react-native";
import { Avatar, Divider } from "react-native-elements";
import { CommentItem, List, SocialCounts } from "..";
import { API, Params } from "../../api/api";
import { UsersPag } from "../../types/api";
import { Comment } from "../../types/comment";
import { Post } from "../../types/post";
import { User } from "../../types/user";
import { formatPostDate } from "../../utils/date";
import { HomeNavigationProps } from "../../utils/navigation";
import { FetchItemsResponse } from "../List";
import { Pressable } from "../Pressable";
import { Reply } from "./Reply";

interface Props {
	post_id: string,
	event_id: string,
	event_name: string,
	event_logo_url?: string,
}

/** Postx represents a post from an event. */
export const Postx = (props: Props) => {
	const navigation = useNavigation<HomeNavigationProps>();
	const [post, setPost] = useState<Post>();
	const [commentsCount, setCommentsCount] = useState<number>(0);
	const [refreshComments, setRefreshComments] = useState<boolean>();

	const getComments = async (params?: Params<Comment>): FetchItemsResponse<Comment> => {
		const resp = await API.Events.GetReplies(props.event_id, props.post_id, params);
		setCommentsCount(resp.comments ? commentsCount + resp.comments.length : commentsCount);
		return [resp.next_cursor, resp.comments];
	};

	useEffect(() => {
		const getPost = async () => {
			try {
				const p = await API.Events.GetPost(props.event_id, props.post_id);
				setPost(p);
			} catch (err) {
				console.log(err);
			}
		};
		getPost();
	}, [props.event_id, props.post_id]);

	const getLikes = async (params?: Params<User>): FetchItemsResponse<User> => {
		const resp = await API.Events.GetPostLikes<UsersPag>(props.event_id, props.post_id, params);
		return [resp.next_cursor, resp.users];
	};

	const goToEvent = () => navigation.navigate("Event", { event_id: props.event_id });

	const Postt = () => (
		<View style={styles.postContainer}>
			<View style={styles.header}>
				<Pressable onPress={goToEvent}>
					<Avatar rounded size={45} source={{ uri: props.event_logo_url }} />
				</Pressable>
				<View style={styles.title}>
					<Pressable onPress={goToEvent}>
						<Text style={styles.name}>{props.event_name}</Text>
					</Pressable>
				</View>
			</View>

			<Text style={styles.content}>{post?.content}</Text>
			<Text style={styles.createdAt}>{formatPostDate(post ? post.created_at : new Date())}</Text>

			<Divider width={0.8} style={styles.divider} />

			{post &&
				<View>
					<SocialCounts
						likes_count={post.likes_count}
						comment_count={post.comments_count}
						liked={post.auth_user_liked}
						like={() => API.Events.LikePost(props.event_id, post.id)}
						getUsersLikes={(params) => getLikes(params)}
					/>
					<Divider style={styles.divider} />
					<Reply
						event_id={props.event_id}
						post_id={post.id}
						onSubmit={() => setRefreshComments(!refreshComments)}
					/>
				</View>
			}
		</View>
	);

	return (
		<View style={styles.container}>
			<View>
				<List<Comment>
					fetchItems={(params) => getComments(params)}
					renderItem={({ item }) => <CommentItem event_id={props.event_id} comment={item} />}
					ListHeaderComponent={Postt}
					extraData={refreshComments}
				/>
			</View>
		</View>
	);
};

const { width } = Dimensions.get("screen");

const styles = StyleSheet.create({
	container: {
		backgroundColor: "white",
		borderBottomWidth: 0.2,
		borderBottomColor: "rgba(0,0,0,0.2)",
	},
	postContainer: {
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
	addComment: {
		flex: 0.1,
		flexDirection: "row",
	},
});
