import React, { useContext, useEffect, useState } from "react";
import { StyleSheet } from "react-native";
import { Card, Text } from "react-native-elements";
import { API, Params } from "../../api/api";
import { Center, List, Notificationx } from "../../components";
import { SessionContext } from "../../context/Session";
import { NotificationsCount, NotificationsPag } from "../../types/api";
import { Notification } from "../../types/notification";

export const App = () => {
	const { user } = useContext(SessionContext);
	const [notificationsCount, setNotificationsCount] = useState<number>();

	const getNotifications = async (params?: Params<Notification>): Promise<[string, Notification[]] | undefined> => {
		if (user) {
			const resp = await API.Notifications.GetFromUser<NotificationsPag>(user.id, params);
			return [resp.next_cursor, resp.notifications];
		}
	};

	const getNotificationsCount = async () => {
		if (user) {
			try {
				const res = await API.Notifications.GetFromUser<NotificationsCount>(user.id, { count: true });
				setNotificationsCount(res.notifications_count);
			} catch (err) {
				console.log(err);
			}
		}
	};

	useEffect(() => {
		getNotificationsCount();
	});

	if (!notificationsCount || notificationsCount === 0) {
		return (
			<Center>
				<Text style={styles.emptyMsg}>
					No notifications
				</Text>
			</Center>
		);
	}

	return (
		<Card containerStyle={styles.card}>
			<List<Notification>
				fetchItems={(params) => getNotifications(params)}
				renderItem={({ item }) => <Notificationx notification={item} />}
			/>
		</Card>
	);
};

const styles = StyleSheet.create({
	card: {
		padding: 0,
	},
	emptyMsg: {
		bottom: 22,
		fontSize: 17,
	},
});
