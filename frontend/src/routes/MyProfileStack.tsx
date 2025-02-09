import { createStackNavigator } from "@react-navigation/stack";
import React, { useContext } from "react";
import { GoBackIcon } from "../components";
import { SessionContext } from "../context/Session";
import { Comment, Event, List, Post, UpdateUser, User } from "../screens";

const { Navigator, Screen } = createStackNavigator();

export const MyProfileStack = () => {
	const { user } = useContext(SessionContext);

	return (
		<Navigator screenOptions={{ headerBackImage: () => GoBackIcon() }}>
			<Screen name="User" component={User}
				initialParams={{ user_id: user?.id }}
				options={{
					headerTitle: "",
					headerTitleAlign: "center",
				}}
			/>
			<Screen name="Comment" component={Comment} />
			<Screen name="Event" component={Event} options={{
				headerTransparent: true,
				headerTitle: "",
			}} />
			<Screen name="Post" component={Post} />
			<Screen name="List" component={List} options={{
				headerTitle: "",
				headerStyle: { elevation: 0 },
			}} />
			{/* TODO: add the other screens to all the stacks */}
			<Screen name="UpdateUser" component={UpdateUser} options={{
				headerTitle: "Edit profile",
			}} />
		</Navigator>
	);
};
