import { useNavigation } from "@react-navigation/core";
import React, { memo, useEffect, useState } from "react";
import { FlatList, StyleSheet, Text, View } from "react-native";
import { Avatar, Card } from "react-native-elements";
import FastImage from "react-native-fast-image";
import { API, Params } from "../../api/api";
import { UsersPag } from "../../types/api";
import { Event } from "../../types/event";
import { Post } from "../../types/post";
import { User } from "../../types/user";
import { formatPostDate } from "../../utils/date";
import { THomeNavigationProps } from "../../utils/navigation";
import { FetchItemsResponse } from "../List";
import { Pressable } from "../Pressable";
import { SocialCounts } from "../SocialCounts";

interface Props {
	post: Post
}

/** PostItem is used to display a list of posts. */
export const PostItem = memo((props: Props) => {
	const { post } = props;
	const [event, setEvent] = useState<Event>();
	const navigation = useNavigation<THomeNavigationProps<User>>();

	useEffect(() => {
		const getEvent = async () => {
			try {
				const e = await API.Events.GetByID(post.event_id);
				setEvent(e);
			} catch (err) {
				console.log(err);
			}
		};

		getEvent();
	}, [post.event_id]);

	const getLikes = async (params?: Params<User>): FetchItemsResponse<User> => {
		if (event) {
			const resp = await API.Events.GetPostLikes<UsersPag>(event.id, props.post.id, params);
			return [resp.next_cursor, resp.users];
		}
	};

	const goToEvent = () => event && navigation.push("Event", { event_id: event.id });

	return (
		<Pressable
			onPress={() => {
				if (event) {
					navigation.push("Post", {
						post_id: post.id,
						event_id: event.id,
						event_name: event.name,
						event_logo_url: event?.logo_url,
					});
				}
			}}
		>
			<Card containerStyle={styles.container}>
				<View style={styles.header}>
					<Pressable onPress={goToEvent}>
						<Avatar rounded size={50} source={{ uri: event?.logo_url }} />
					</Pressable>
					<View style={styles.title}>
						<Pressable onPress={goToEvent}>
							<Text style={styles.name}>{event?.name}</Text>
						</Pressable>
						<Text style={styles.createdAt}>{formatPostDate(post.created_at)}</Text>
					</View>
				</View>

				<Text style={styles.content}>{post.content}</Text>
				{/**
				 * TODO: display a grid of images (max 2, indicating how many more there are) and show a modal that
				 * allows to swipe between them when one is clicked (the list should start with the index of the image clicked)
				*/}
				<FlatList
					data={post.media}
					keyExtractor={item => item}
					renderItem={({ item }) => <FastImage source={{ uri: item }} style={styles.image} />}
				// horizontal -> somehow breaks the image
				/>

				<SocialCounts
					comment_count={post.comments_count}
					likes_count={post.likes_count}
					liked={post.auth_user_liked}
					like={() => API.Events.LikePost(event ? event.id : "", post.id)}
					getUsersLikes={(params) => getLikes(params)}
				/>
			</Card>
		</Pressable>
	);
});

const styles = StyleSheet.create({
	container: {
		margin: 0,
		padding: 10,
	},
	header: {
		flexDirection: "row",
		alignItems: "center",
	},
	title: {
		marginLeft: 12,
	},
	name: {
		fontSize: 20,
		fontWeight: "bold",
	},
	createdAt: {
		fontSize: 15,
	},
	content: {
		marginVertical: 10,
		marginHorizontal: 5,
		fontSize: 18,
	},
	image: {
		width: "102%",
		height: 200,
		alignSelf: "center",
		marginBottom: 10,
	},
});
