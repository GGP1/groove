import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { useNavigation } from "@react-navigation/native";
import React, { useContext, useEffect, useState } from "react";
import { StyleSheet } from "react-native";
import { Avatar, Icon, withBadge } from "react-native-elements";
import { API } from "../api/api";
import { SessionContext } from "../context/Session";
import { NotificationsCount } from "../types/api";
import { HomeNavigationProps } from "../utils/navigation";
import { Notifications } from "../utils/notifications";
import { EventsStack } from "./EventsStack";
import { ExploreStack } from "./ExploreStack";
import { HomeStack } from "./HomeStack";
import { MyProfileStack } from "./MyProfileStack";
import { NotificationsStack } from "./NotificationsStack";

const { Navigator, Screen } = createBottomTabNavigator();
const ICON_SIZE = 31;

export const Tabs = () => {
	const navigation = useNavigation<HomeNavigationProps>();
	const { user } = useContext(SessionContext);
	const [initialRoute, setInitialRoute] = useState<string>();
	const [notificationsCount, setNotificationsCount] = useState<number>(0);

	const NotificationsIcon = withBadge(notificationsCount, {
		badgeStyle: styles.badgeStyle,
		hidden: notificationsCount === 0,
	})(({ color }) => <Icon name="notifications" size={ICON_SIZE} color={color} />);

	useEffect(() => {
		const getNotifCount = async () => {
			if (user) {
				try {
					const res = await API.Notifications.GetFromUser<NotificationsCount>(user.id, { count: true });
					setNotificationsCount(res.notifications_count);
				} catch (err) {
					console.log(err);
				}
			}
		};

		Notifications.onMessageBackground();
		Notifications.onMessageForeground(() => setNotificationsCount(notificationsCount + 1));
		Notifications.onNotificationOpenedApp(navigation);
		// Check whether an initial notification is available
		Notifications.getInitialNotification(message => setInitialRoute(message.data?.screen));
		getNotifCount();
	}, [navigation, notificationsCount, user]);

	return (
		<Navigator
			initialRouteName={initialRoute}
			screenOptions={({ route }) => ({
				tabBarActiveTintColor: "crimson",
				tabBarInactiveTintColor: "gray",
				tabBarHideOnKeyboard: true,
				tabBarShowLabel: false,
				tabBarIcon: ({ color }) => <Icon name={iconName(route.name)} color={color} size={ICON_SIZE} />,
				headerShown: false,
			})}
		>
			<Screen name="HomeStack" component={HomeStack} />
			<Screen name="ExploreStack" component={ExploreStack} />
			<Screen name="EventsStack" component={EventsStack} />
			<Screen name="NotificationsStack" component={NotificationsStack}
				listeners={{ tabPress: () => setNotificationsCount(0) }}
				options={{
					headerTitle: "Notifications",
					// @ts-ignore
					tabBarIcon: ({ color }) => <NotificationsIcon color={color} size={ICON_SIZE} />,
				}}
			/>
			<Screen name="MyProfileStack" component={MyProfileStack}
				options={{
					tabBarIcon: ({ color }) => (
						<Avatar
							rounded
							containerStyle={[styles.avatar, { borderColor: color }]}
							size={ICON_SIZE}
							source={user?.profile_image_url ? { uri: user.profile_image_url } : require("../../assets/icons/defUserAvatar.png")}
						/>
					),
				}}
			/>
		</Navigator>
	);
};


const iconName = (routeName: string): string => {
	switch (routeName) {
		case "ExploreStack":
			return "search";
		case "EventsStack":
			return "event";
		case "NotificationsStack":
			return "notifications";
	}

	return "home";
};

const styles = StyleSheet.create({
	badgeStyle: {
		left: -8,
	},
	avatar: {
		borderWidth: 1.5,
	},
});
