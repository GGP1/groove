import { useNavigation } from "@react-navigation/native";
import React, { useState } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { Icon } from "react-native-elements";
import { Params } from "../api/api";
import { User } from "../types/user";
import { THomeNavigationProps } from "../utils/navigation";
import { FetchItemsResponse } from "./List";
import { UserItem } from "./user/UserItem";

interface Props {
	likes_count: number,
	comment_count: number,
	liked: boolean,
	like: () => Promise<void>,
	getUsersLikes: (params?: Params<User>) => FetchItemsResponse<User>
}

const ICON_SIZE = 29;

export const SocialCounts = (props: Props) => {
	const [liked, setLiked] = useState<boolean>(props.liked);
	const [likes, setLikes] = useState<number>(props.likes_count);
	const navigation = useNavigation<THomeNavigationProps<User>>();

	const like = async () => {
		liked ? setLikes(likes - 1) : setLikes(likes + 1);
		setLiked(!liked);
		try {
			await props.like();
		} catch (err) {
			console.log(err);
		}
	};

	return (
		<View style={styles.container}>
			<View style={styles.social}>
				<Icon name="comment-outline" type="material-community" size={ICON_SIZE} />
				<Text style={styles.count}>{props.comment_count}</Text>
			</View>

			<View style={styles.social}>
				{/* TODO: is it possible to perform the like when unmounting the component to prevent from like spamming? */}
				<Pressable onPress={like}>
					{liked ? <Icon name="thumb-up" size={ICON_SIZE} /> : <Icon name="thumb-up-off-alt" size={ICON_SIZE} />}
				</Pressable>

				<Pressable onPress={() => navigation.navigate("List", {
					title: "Likes",
					props: {
						fetchItems: (params) => props.getUsersLikes(params),
						renderItem: ({ item }) => <UserItem user={item} />,
					},
				})}>
					<Text style={styles.count}>{likes}</Text>
				</Pressable>
			</View>
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		flexDirection: "row",
		justifyContent: "flex-end",
	},
	social: {
		flexDirection: "row",
		alignItems: "center",
		marginRight: 15,
	},
	count: {
		marginHorizontal: 5,
		// fontSize: 15,
	},
});
