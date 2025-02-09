import { useNavigation } from "@react-navigation/native";
import React from "react";
import { Dimensions, StyleSheet, Text, View } from "react-native";
import { Avatar, Card } from "react-native-elements";
import { User } from "../../types/user";
import { HomeNavigationProps } from "../../utils/navigation";
import { Pressable } from "../Pressable";

interface Props {
	user: User,
}

const { width } = Dimensions.get("screen");

export const UserItem = (props: Props) => {
	const navigation = useNavigation<HomeNavigationProps>();
	const { user } = props;

	return (
		<Pressable onPress={() => navigation.push("User", { user_id: props.user.id })}>
			<Card containerStyle={styles.container}>
				<View style={styles.header}>
					<Avatar
						rounded
						size={55}
						source={user?.profile_image_url ? { uri: user.profile_image_url } : require("../../../assets/icons/defUserAvatar.png")}
					/>
					<View style={styles.info}>
						<Text style={styles.name}>{user?.name}</Text>
						<Text style={styles.username}>{user?.username}</Text>
					</View>
				</View>
			</Card>
		</Pressable>
	);
};

const styles = StyleSheet.create({
	container: {
		margin: 0,
		width: width,
	},
	header: {
		flexDirection: "row",
		marginBottom: 8,
	},
	info: {
		marginLeft: 15,
	},
	name: {
		fontSize: 20,
		fontWeight: "bold",
	},
	username: {
		fontSize: 18,
	},
	content: {
		marginHorizontal: 5,
		marginBottom: 10,
	},
});
