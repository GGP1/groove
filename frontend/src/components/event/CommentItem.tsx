import { useNavigation } from "@react-navigation/native";
import React, { memo, useEffect, useState } from "react";
import { Dimensions, StyleSheet, Text, View } from "react-native";
import { Avatar, Card } from "react-native-elements";
import { API, Params } from "../../api/api";
import { UsersPag } from "../../types/api";
import { Comment } from "../../types/comment";
import { User } from "../../types/user";
import { HomeNavigationProps } from "../../utils/navigation";
import { FetchItemsResponse } from "../List";
import { Pressable } from "../Pressable";
import { SocialCounts } from "../SocialCounts";

interface Props {
	event_id: string,
	comment: Comment,
}

/** CommentItem is the component used to display comments inside a list, usually replies. */
export const CommentItem = memo((props: Props) => {
	const { comment } = props;
	const [user, setUser] = useState<User>();
	const navigation = useNavigation<HomeNavigationProps>();

	useEffect(() => {
		const getUser = async () => {
			try {
				const u = await API.Users.GetByID(comment.user_id);
				setUser(u);
			} catch (err) {
				console.log(err);
			}
		};

		getUser();
	}, [comment]);

	const getLikes = async (params?: Params<User>): FetchItemsResponse<User> => {
		const resp = await API.Events.GetCommentLikes<UsersPag>(props.event_id, props.comment.id, params);
		return [resp.next_cursor, resp.users];
	};

	return (
		<Pressable onPress={() => user && navigation.push("Comment", {
			comment_id: props.comment.id,
			event_id: props.event_id,
			user_id: user.id,
		})}>
			<Card containerStyle={styles.container}>
				<View style={styles.header}>
					<Avatar
						rounded
						size={50}
						source={user?.profile_image_url ? { uri: user.profile_image_url } : require("../../../assets/icons/defUserAvatar.png")}
					/>
					<View style={styles.who}>
						<Text style={styles.name}>{user?.name}</Text>
						<Text style={styles.username}>{user?.username}</Text>
					</View>
				</View>
				<Text style={styles.content}>{comment?.content}</Text>
				{comment &&
					<SocialCounts
						likes_count={comment.likes_count}
						comment_count={comment.replies_count}
						liked={comment.auth_user_liked}
						like={() => API.Events.LikeComment(props.event_id, comment.id)}
						getUsersLikes={(params) => getLikes(params)}
					/>
				}
			</Card>
		</Pressable>
	);
});

const { width } = Dimensions.get("screen");

const styles = StyleSheet.create({
	container: {
		margin: 0,
		width: width,
	},
	header: {
		flexDirection: "row",
		marginBottom: 8,
		alignItems: "center",
	},
	who: {
		marginLeft: 10,
	},
	name: {
		fontSize: 19,
		fontWeight: "bold",
	},
	username: {
		fontSize: 16,
	},
	content: {
		fontSize: 18,
		marginHorizontal: 5,
		marginBottom: 10,
	},
});
