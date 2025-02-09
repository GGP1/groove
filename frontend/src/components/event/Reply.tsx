import React, { useContext, useState } from "react";
import { Dimensions, StyleSheet, TextInput, View } from "react-native";
import { Avatar, Icon } from "react-native-elements";
import { API } from "../../api/api";
import { SessionContext } from "../../context/Session";
import { CreateComment } from "../../types/comment";

interface Props {
	post_id?: string,
	parent_comment_id?: string,
	event_id: string,
	onSubmit: () => void
}

// TODO: consider using a comment like instagram (add comment section as footer)
// or like twitter (new screen but below the parent)

/** Reply is the component used below posts and comments that have an input box for the
users to leave comments. It should receive either a post ID or a comment ID in the properties. */
export const Reply = (props: Props) => {
	const { user } = useContext(SessionContext);
	const [content, setContent] = useState<string>("");

	const reply = async () => {
		const cc: CreateComment = {
			content: content,
			post_id: props.post_id,
			parent_comment_id: props.parent_comment_id,
		};
		await API.Events.CreateComment(props.event_id, cc);
		props.onSubmit();
		setContent("");
	};

	return (
		<View style={styles.container}>
			<Avatar
				rounded
				size={45}
				source={user?.profile_image_url ? { uri: user.profile_image_url } : require("../../../assets/icons/defUserAvatar.png")}
			/>
			<TextInput
				style={styles.input}
				placeholder="Add a comment..."
				value={content}
				onChangeText={text => setContent(text)}
				multiline
			/>
			<Icon
				containerStyle={styles.icon}
				name="send"
				size={30}
				onPress={reply}
				color="crimson"
			/>
		</View>
	);
};

const { width } = Dimensions.get("screen");

const styles = StyleSheet.create({
	container: {
		margin: 10,
		flexDirection: "row",
		alignItems: "center",
		alignSelf: "center",
	},
	input: {
		width: width - 105,
		backgroundColor: "rgba(0, 0, 0, 0.1)",
		borderRadius: 5,
		marginLeft: 10,
		fontSize: 18,
	},
	icon: {
		marginLeft: 7,
	},
});
