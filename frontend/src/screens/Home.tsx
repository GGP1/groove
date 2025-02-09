import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { Divider } from "react-native-elements";
// import { SafeAreaView } from "react-native-safe-area-context";
import { API, Params } from "../api/api";
import { List, PostItem } from "../components";
import { FetchItemsResponse } from "../components/List";
import { Post } from "../types/post";

export const App = () => {
	const getHomePosts = async (params?: Params<Post>): FetchItemsResponse<Post> => {
		const resp = await API.Events.GetHomePosts(params);
		return [resp.next_cursor, resp.posts];
	};

	return (
		<View style={styles.container}>
			<Text style={styles.title}>
				Home
			</Text>

			<Divider />

			<List<Post>
				fetchItems={(params) => getHomePosts(params)}
				renderItem={({ item }) => <PostItem post={item} />}
			/>
		</View>
	);
};

const styles = StyleSheet.create({
	container: {
		flex: 1,
		backgroundColor: "white",
	},
	title: {
		padding: 5,
		fontSize: 17,
		fontWeight: "bold",
	},
});
