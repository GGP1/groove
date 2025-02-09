import { createStackNavigator } from "@react-navigation/stack";
import React from "react";
import { GoBackIcon, Map } from "../components";
import { Comment, Event, Explore, List, Post, User } from "../screens";

const { Navigator, Screen } = createStackNavigator();

export const ExploreStack = () => {
	return (
		<Navigator screenOptions={{ headerBackImage: () => GoBackIcon() }}>
			<Screen name="Explore" component={Explore} options={{
				headerShown: false,
			}} />
			<Screen name="Comment" component={Comment} />
			<Screen name="Event" component={Event} options={{
				headerTransparent: true,
				headerTitle: "",
			}} />
			<Screen name="Map" component={Map} options={{
				headerShown: true,
				headerTransparent: true,
				headerTitle: "",
				headerTintColor: "white",
				headerBackImage: () => GoBackIcon("mintcream"),
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
