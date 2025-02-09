import { useNavigation } from "@react-navigation/core";
import React, { useEffect, useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Avatar } from "react-native-elements";
import { API } from "../api/api";
import { Notification } from "../types/notification";
import { User } from "../types/user";
import { HomeNavigationProps } from "../utils/navigation";

interface Props {
	notification: Notification,
}

export const Notificationx = (props: Props) => {
	const navigation = useNavigation<HomeNavigationProps>();
	const [sender, setSender] = useState<User>();

	useEffect(() => {
		const getSender = async () => {
			try {
				const u = await API.Users.GetByID(props.notification.sender_id);
				setSender(u);
			} catch (err) {
				console.log(err);
			}
		};

		getSender();
	});

	return (
		<View style={styles.container}>
			<Avatar
				rounded
				source={{ uri: sender?.profile_image_url }}
				onPress={() => sender ? navigation.navigate("User", { user_id: sender.id }) : null}
			/>
			<Text style={styles.title}>
				{sender?.username}
			</Text>
			<Text style={styles.body}>
				Notification ID: {props.notification.id + "\n"}
				Sender ID: {props.notification.sender_id + "\n"}
				Receiver ID: {props.notification.receiver_id + "\n"}
				{/* {props.notification.event_id && <Text>Event ID: {props.notification.event_id}</Text>} */}
				Content: {props.notification.content + "\n"}
				Created at: {props.notification.created_at + "\n"}
			</Text>
		</View>
	);
};


const styles = StyleSheet.create({
	container: {
		flex: 1,
		flexDirection: "row",
	},
	title: {
		fontSize: 15,
	},
	body: {
		fontSize: 13,
	},
});
