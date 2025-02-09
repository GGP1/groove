import { createStackNavigator } from "@react-navigation/stack";
import React from "react";
import { GoBackIcon } from "../components";
import { Comment, Event, List, Notifications, Post, User } from "../screens";

const { Navigator, Screen } = createStackNavigator();

export const NotificationsStack = () => {
	return (
		<Navigator screenOptions={{ headerBackImage: () => GoBackIcon() }}>
			<Screen name="Notifications" component={Notifications} />
			<Screen name="Comment" component={Comment} />
			<Screen name="Event" component={Event} options={{
				headerTransparent: true,
				headerTitle: "",
			}} />
			<Screen name="Post" component={Post} />
			<Screen name="User" component={User} options={{
				headerTitle: "",
				headerTitleAlign: "center",
			}} />
			<Screen name="List" component={List} options={{
				headerTitle: "",
				headerStyle: { elevation: 0 },
			}} />
		</Navigator>
	);
};
